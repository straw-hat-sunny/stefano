package chat

import (
	"context"

	"github.com/google/uuid"
)

const assistantMessage = "Welcome, I am Stefano your personal assistant. How can I assist you today?"
const assistantUser = "assistant"

// Message is a message in a chat session.
type Message struct {
	ID      uuid.UUID `json:"id"`      // uuidv7
	User    string    `json:"user"`    // "user" or "assistant"
	Content string    `json:"content"` // The message content
}

func NewMessage(user string, content string) (Message, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Message{}, err
	}
	return Message{ID: id, User: user, Content: content}, nil
}

type llmClient interface {
	GenerateMessage(ctx context.Context, userMessage string) (string, error)
}

type Chat struct {
	ID        uuid.UUID // uuidv7
	Messages  []Message
	llmClient llmClient
}

func (c *Chat) generateAssistantMessage(ctx context.Context, userMessage Message) (Message, error) {
	assistantMessage, err := c.llmClient.GenerateMessage(ctx, userMessage.Content)
	if err != nil {
		return Message{}, err
	}
	return NewMessage(assistantUser, assistantMessage)
}
