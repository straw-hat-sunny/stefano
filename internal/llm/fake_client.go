package llm

import (
	"context"
	"time"
)

type FakeLLMClient struct{}

func NewFakeLLMClient() *FakeLLMClient {
	return &FakeLLMClient{}
}

func (c *FakeLLMClient) GenerateMessage(ctx context.Context, userMessage string) (string, error) {
	time.Sleep(3 * time.Second)
	return "Thinking...", nil
}
