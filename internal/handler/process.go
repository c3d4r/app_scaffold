package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	ltypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/c3d4r/app_scaffold/internal/models"
	"github.com/c3d4r/app_scaffold/internal/store"
)

func LambdaProcessStarter(client *lambda.Client, functionName string) ProcessStarter {
	return func(chatID, msgID string) error {
		payload, err := json.Marshal(map[string]string{
			"chatId": chatID,
			"msgId":  msgID,
		})
		if err != nil {
			return err
		}
		_, err = client.Invoke(context.Background(), &lambda.InvokeInput{
			FunctionName:   aws.String(functionName),
			InvocationType: ltypes.InvocationTypeEvent,
			Payload:        payload,
		})
		return err
	}
}

func InlineProcessStarter(chatStore store.ChatStore, bedrockClient *bedrockruntime.Client, modelID string) ProcessStarter {
	return func(chatID, msgID string) error {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := processInDev(ctx, chatStore, bedrockClient, modelID, chatID, msgID); err != nil {
				log.Printf("process error: %v", err)
			}
		}()
		return nil
	}
}

func processInDev(ctx context.Context, chatStore store.ChatStore, bedrockClient *bedrockruntime.Client, modelID, chatID, msgID string) error {
	chat, err := chatStore.GetChat(ctx, chatID)
	if err != nil || chat == nil {
		return err
	}

	var messages []brtypes.Message
	for _, m := range chat.Messages {
		if m.Status == "processing" {
			continue
		}
		contentBlocks, err := buildConverseContentBlocks(ctx, chatStore, m)
		if err != nil {
			log.Printf("WARN build content blocks for msg %s: %v", m.ID, err)
			contentBlocks = []brtypes.ContentBlock{
				&brtypes.ContentBlockMemberText{Value: m.Content},
			}
		}
		if len(contentBlocks) == 0 {
			continue
		}
		messages = append(messages, brtypes.Message{
			Role:    toConversationRole(m.Role),
			Content: contentBlocks,
		})
	}

	resp, err := bedrockClient.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId:  aws.String(modelID),
		Messages: messages,
		InferenceConfig: &brtypes.InferenceConfiguration{
			MaxTokens:   aws.Int32(1024),
			Temperature: aws.Float32(0.7),
		},
	})
	if err != nil {
		log.Printf("bedrock converse error: %v", err)
		return err
	}

	text := extractResponseText(resp)
	fragment := buildFragmentHTML(text)

	if err := chatStore.PutFragment(ctx, chatID, msgID, fragment); err != nil {
		log.Printf("put fragment error: %v", err)
		return err
	}

	for i := range chat.Messages {
		if chat.Messages[i].ID == msgID {
			chat.Messages[i].Status = "complete"
			chat.Messages[i].Content = text
			chat.Messages[i].Fragment = "messages/" + chatID + "/" + msgID + ".html"
			break
		}
	}

	if err := chatStore.SaveChat(ctx, chat); err != nil {
		log.Printf("save chat error: %v", err)
		return err
	}

	return nil
}

func buildConverseContentBlocks(ctx context.Context, chatStore store.ChatStore, msg models.Message) ([]brtypes.ContentBlock, error) {
	var blocks []brtypes.ContentBlock

	if msg.Content != "" {
		blocks = append(blocks, &brtypes.ContentBlockMemberText{Value: msg.Content})
	}

	for _, att := range msg.Attachments {
		data, err := chatStore.GetFile(ctx, att.Key)
		if err != nil {
			log.Printf("WARN skip attachment %s: %v", att.Key, err)
			continue
		}
		if len(data) == 0 {
			log.Printf("WARN skip empty attachment %s", att.Key)
			continue
		}

		switch {
		case strings.HasPrefix(att.Type, "image/"):
			format := strings.TrimPrefix(att.Type, "image/")
			if format == "jpeg" {
				format = "jpeg"
			}
			blocks = append(blocks, &brtypes.ContentBlockMemberImage{
				Value: brtypes.ImageBlock{
					Format: brtypes.ImageFormat(format),
					Source: &brtypes.ImageSourceMemberBytes{
						Value: data,
					},
				},
			})
		case att.Type == "application/pdf":
			blocks = append(blocks, &brtypes.ContentBlockMemberDocument{
				Value: brtypes.DocumentBlock{
					Name:   &att.Name,
					Format: brtypes.DocumentFormatPdf,
					Source: &brtypes.DocumentSourceMemberBytes{
						Value: data,
					},
				},
			})
		case strings.HasPrefix(att.Type, "text/"):
			text := fmt.Sprintf("[File: %s]\n%s", att.Name, string(data))
			blocks = append(blocks, &brtypes.ContentBlockMemberText{Value: text})
		}
	}

	return blocks, nil
}

func EchoProcessStarter(chatStore store.ChatStore) ProcessStarter {
	return func(chatID, msgID string) error {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			time.Sleep(1 * time.Second)
			text := "Hello! This is a development fallback response.\n\nSet up AWS credentials and Bedrock model access to use a real LLM."
			fragment := buildFragmentHTML(text)

			if err := chatStore.PutFragment(ctx, chatID, msgID, fragment); err != nil {
				log.Printf("echo put fragment error: %v", err)
				return
			}

			chat, _ := chatStore.GetChat(ctx, chatID)
			if chat != nil {
				for i := range chat.Messages {
					if chat.Messages[i].ID == msgID {
						chat.Messages[i].Status = "complete"
						chat.Messages[i].Content = text
						chat.Messages[i].Fragment = "messages/" + chatID + "/" + msgID + ".html"
						break
					}
				}
				chatStore.SaveChat(ctx, chat)
			}
		}()
		return nil
	}
}

func toConversationRole(role string) brtypes.ConversationRole {
	switch role {
	case "user":
		return brtypes.ConversationRoleUser
	case "assistant":
		return brtypes.ConversationRoleAssistant
	default:
		return brtypes.ConversationRoleUser
	}
}

func extractResponseText(resp *bedrockruntime.ConverseOutput) string {
	if resp == nil {
		return ""
	}
	switch output := resp.Output.(type) {
	case *brtypes.ConverseOutputMemberMessage:
		for _, block := range output.Value.Content {
			switch content := block.(type) {
			case *brtypes.ContentBlockMemberText:
				return content.Value
			}
		}
	}
	return ""
}

func buildFragmentHTML(text string) []byte {
	escaped := html.EscapeString(text)
	escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")
	s := "<div id=\"msg-container\" class=\"flex justify-start\">\n" +
		"  <div class=\"bg-gray-100 text-gray-800 rounded-2xl rounded-bl-md px-4 py-2.5 max-w-[80%] shadow-sm\">\n" +
		"    " + escaped + "\n" +
		"  </div>\n" +
		"</div>\n"
	return []byte(s)
}
