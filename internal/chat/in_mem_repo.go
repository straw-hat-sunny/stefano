package chat

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var ErrChatNotFound = errors.New("chat not found")

type InMemRepo struct {
	mu    sync.Mutex
	chats map[uuid.UUID]Chat
}

func NewInMemRepo() *InMemRepo {
	return &InMemRepo{chats: make(map[uuid.UUID]Chat)}
}

func (r *InMemRepo) CreateChat(ctx context.Context, chat Chat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chats[chat.ID] = chat
	return nil
}

func (r *InMemRepo) GetChat(ctx context.Context, id uuid.UUID) (Chat, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	chat, ok := r.chats[id]
	if !ok {
		return Chat{}, ErrChatNotFound
	}
	return chat, nil
}

func (r *InMemRepo) UpdateChat(ctx context.Context, chat Chat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chats[chat.ID] = chat
	return nil
}

func (r *InMemRepo) DeleteChat(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.chats, id)
	return nil
}
