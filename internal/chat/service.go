package chat

import (
	"context"

	"github.com/google/uuid"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, chat Chat) error
	GetChat(ctx context.Context, id uuid.UUID) (Chat, error)
	UpdateChat(ctx context.Context, chat Chat) error
	DeleteChat(ctx context.Context, id uuid.UUID) error
}

// Service is the chat service, responsible for managing the chat sessions and messages.
type Service struct {
	repo ChatRepository
}

// NewService returns a new chat service.
func NewService(repo ChatRepository) *Service {
	return &Service{repo: repo}
}

// CreateChat creates a new chat session and returns the chat object.
func (s *Service) CreateChat(ctx context.Context) (Chat, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Chat{}, err
	}

	message, err := NewMessage(assistantUser, assistantMessage)
	if err != nil {
		return Chat{}, err
	}

	chat := Chat{
		ID:       id,
		Messages: []Message{message},
	}
	err = s.repo.CreateChat(ctx, chat)
	if err != nil {
		return Chat{}, err
	}

	return chat, nil
}

// GetChat returns a chat by id.
func (s *Service) GetChat(ctx context.Context, id uuid.UUID) (Chat, error) {
	return s.repo.GetChat(ctx, id)
}

// ProcessMessage is the function that processes the user message and returns the assistant message.
func (s *Service) ProcessMessage(ctx context.Context, id uuid.UUID, userMessage Message) (Message, error) {
	chat, err := s.repo.GetChat(ctx, id)
	if err != nil {
		return Message{}, err
	}

	// Generate a message from the assistant
	assistantMessage, err := NewMessage(assistantUser, "Thinking...")
	if err != nil {
		return Message{}, err
	}

	// update the chat with the new messages
	chat.Messages = append(chat.Messages, userMessage, assistantMessage)
	err = s.repo.UpdateChat(ctx, chat)
	if err != nil {
		return Message{}, err
	}

	return assistantMessage, nil
}
