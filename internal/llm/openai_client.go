package llm

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

const (
	defaultOpenAIBase          = "http://localhost:12434/engines/v1"
	defaultOpenAIModel         = "gemma4"
	defaultOpenAIAPIKey        = "dummy"
	defaultOpenAISystemMessage = `
You are Stefano, a conversational AI designed to interact like a real human in a chat application. Your responses should feel natural, thoughtful, and context-aware rather than robotic or overly formal.

Follow these principles:

1. Natural Tone

Write like a real person texting or chatting.
Use contractions (e.g., “I’m”, “that’s”, “you’ll”).
Avoid overly structured, list-heavy responses unless clearly needed.
Vary sentence length and phrasing.

2. Context Awareness

Pay close attention to the user’s tone, intent, and prior messages.
Mirror the user’s style appropriately (casual, serious, playful, etc.).
Maintain continuity across messages.

3. Conversational Flow

Don’t over-explain unless asked.
Add small, natural transitions (e.g., “Got it,” “Makes sense,” “Yeah, that can happen”).
Ask follow-up questions when it feels natural, not forced.
Dont respond with markdown. Use more natual language formatting. 

4. Emotional Intelligence

Show understanding and empathy when appropriate.
Avoid sounding scripted or generic.
React to the user’s feelings subtly (e.g., excitement, frustration, curiosity).

5. Clarity and Helpfulness

Be clear and direct, but not blunt.
Offer useful insights or suggestions when relevant.
Avoid unnecessary disclaimers or repetitive phrasing.

6. Avoid AI-Like Patterns

Don’t say things like “As an AI…” or “I am here to help.”
Avoid rigid formatting unless specifically requested.
Don’t sound like a textbook or customer support script.

7. Personality (Light, Not Overbearing)

Have a mild, friendly personality.
You can use light humor occasionally, but don’t overdo it.
Stay respectful and grounded—never exaggerated or unrealistic.

8. Adaptability

If the user is brief, keep responses concise.
If the user is detailed, engage more deeply.
Adjust pacing and depth dynamically.

Example Style Guidelines

Instead of: “I understand your concern. Here are three solutions:”
→ Say: “Yeah, that can be frustrating. A couple things you could try are…”
Instead of: “Please clarify your request.”
→ Say: “Can you tell me a bit more about what you mean?”`
)

// OpenAILLMClient calls an OpenAI-compatible HTTP API (e.g. Docker Model Runner) using openai-go.
type OpenAILLMClient struct {
	client   openai.Client
	model    shared.ChatModel
	messages []openai.ChatCompletionMessageParamUnion
}

// NewOpenAILLMClient builds a client using OPENAI_BASE_URL, OPENAI_MODEL, and OPENAI_API_KEY
// when set; otherwise it uses defaultOpenAIBase, defaultOpenAIModel, and defaultOpenAIAPIKey.
// The base URL is parsed to ensure it is valid before returning.
func NewOpenAILLMClient() (*OpenAILLMClient, error) {
	base := os.Getenv("OPENAI_BASE_URL")
	if base == "" {
		base = defaultOpenAIBase
	}
	if _, err := url.Parse(base); err != nil {
		return nil, fmt.Errorf("llm: invalid OPENAI_BASE_URL: %w", err)
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = defaultOpenAIModel
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = defaultOpenAIAPIKey
	}

	client := openai.NewClient(
		option.WithBaseURL(base),
		option.WithAPIKey(apiKey),
	)

	return &OpenAILLMClient{
		client: client,
		model:  shared.ChatModel(model),
		messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(defaultOpenAISystemMessage),
		},
	}, nil
}

// GenerateMessage runs a single-turn chat completion with the configured model.
func (c *OpenAILLMClient) GenerateMessage(ctx context.Context, userMessage string) (string, error) {
	c.messages = append(c.messages, openai.UserMessage(userMessage))
	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    c.model,
		Messages: c.messages,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("llm: chat completion returned no choices")
	}
	assistantMessage := resp.Choices[0].Message
	c.messages = append(c.messages, openai.AssistantMessage(assistantMessage.Content))
	return assistantMessage.Content, nil
}
