package models

type Message struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Fragment string `json:"fragment,omitempty"`
}

type Chat struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
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
