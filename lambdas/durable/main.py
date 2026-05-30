import base64
import json
import html
import os
import traceback
from io import BytesIO

import boto3

GENERATED_BUCKET = os.environ["GENERATED_BUCKET"]
BEDROCK_MODEL_ID = os.environ.get(
    "BEDROCK_MODEL_ID", "us.anthropic.claude-sonnet-4-5-20250929-v1:0"
)

s3 = boto3.client("s3")
bedrock_runtime = boto3.client("bedrock-runtime")


def handler(event, context):
    chat_id = event["chatId"]
    msg_id = event["msgId"]

    try:
        process_message(chat_id, msg_id)
    except Exception:
        print(f"Error processing {chat_id}/{msg_id}:")
        traceback.print_exc()


def process_message(chat_id: str, msg_id: str):
    chat = load_chat(chat_id)

    conversation = build_conversation(chat)

    try:
        response = bedrock_runtime.converse(
            modelId=BEDROCK_MODEL_ID,
            messages=conversation,
            inferenceConfig={"maxTokens": 1024, "temperature": 0.7},
        )
        text = response["output"]["message"]["content"][0]["text"]
    except Exception:
        traceback.print_exc()
        text = "Sorry, something went wrong."

    fragment_html = build_fragment(text)

    fragment_key = f"messages/{chat_id}/{msg_id}.html"
    s3.put_object(
        Bucket=GENERATED_BUCKET,
        Key=fragment_key,
        Body=fragment_html.encode("utf-8"),
        ContentType="text/html",
    )

    for msg in chat["messages"]:
        if msg["id"] == msg_id:
            msg["status"] = "complete"
            msg["content"] = text
            msg["fragment"] = fragment_key
            break

    save_chat(chat)


def load_chat(chat_id: str) -> dict:
    key = f"chats/{chat_id}.json"
    resp = s3.get_object(Bucket=GENERATED_BUCKET, Key=key)
    return json.loads(resp["Body"].read())


def save_chat(chat: dict):
    key = f"chats/{chat['id']}.json"
    body = json.dumps(chat, indent=2).encode("utf-8")
    s3.put_object(Bucket=GENERATED_BUCKET, Key=key, Body=body, ContentType="application/json")


def build_conversation(chat: dict) -> list[dict]:
    messages = []
    for msg in chat["messages"]:
        if msg.get("status") == "processing":
            continue
        role = msg["role"]
        content_blocks = []
        if msg.get("content"):
            content_blocks.append({"text": msg["content"]})
        for att in msg.get("attachments", []):
            key = att["key"]
            try:
                data = read_file(key)
                if not data:
                    print(f"Attachment file is empty, skipping: {key}")
                    continue
                mime_type = att["type"]
                if mime_type.startswith("image/"):
                    fmt = _image_format(mime_type)
                    content_blocks.append({
                        "image": {
                            "format": fmt,
                            "source": {"bytes": data},
                        }
                    })
                elif mime_type == "application/pdf":
                    content_blocks.append({
                        "document": {
                            "format": "pdf",
                            "name": att["name"],
                            "source": {"bytes": data},
                        }
                    })
                elif mime_type.startswith("text/") or mime_type == "text/markdown":
                    text = data.decode("utf-8", errors="replace")
                    content_blocks.append({
                        "text": f"[File: {att['name']}]\n{text}"
                    })
            except Exception:
                print(f"Failed to read attachment {key}:")
                traceback.print_exc()

        if content_blocks:
            messages.append({"role": role, "content": content_blocks})
    return messages


def _image_format(mime_type: str) -> str:
    fmt = mime_type.split("/")[-1]
    if fmt in ("jpeg", "png", "gif", "webp"):
        return fmt
    return "jpeg"


def read_file(key: str) -> bytes:
    resp = s3.get_object(Bucket=GENERATED_BUCKET, Key=key)
    return resp["Body"].read()


def build_fragment(text: str) -> str:
    escaped = html.escape(text)
    escaped = escaped.replace("\n", "<br>\n")
    return (
        '<div class="flex justify-start">\n'
        '  <div class="bg-gray-100 text-gray-800 rounded-2xl rounded-bl-md px-4 py-2.5 max-w-[80%] shadow-sm">\n'
        f"    {escaped}\n"
        "  </div>\n"
        "</div>\n"
    )
