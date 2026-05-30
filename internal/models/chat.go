package models

type Attachment struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Size         int64  `json:"size"`
	Key          string `json:"key"`
	PreSignedURL string `json:"-"`
}

type Message struct {
	ID          string       `json:"id"`
	Role        string       `json:"role"`
	Content     string       `json:"content"`
	Status      string       `json:"status"`
	Fragment    string       `json:"fragment,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Chat struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
}

type ChatSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updatedAt"`
}

func NewChat(id string) *Chat {
	return &Chat{
		ID:       id,
		Messages: []Message{},
	}
}

func (c *Chat) AddMessage(m Message) {
	c.Messages = append(c.Messages, m)
}

func (c *Chat) LastAssistant() *Message {
	for i := len(c.Messages) - 1; i >= 0; i-- {
		if c.Messages[i].Role == "assistant" {
			return &c.Messages[i]
		}
	}
	return nil
}
