package chat

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const defaultOpenAIBase = "http://localhost:12434/engines/v1"

// OpenAIClient calls the OpenAI Chat Completions API via the official SDK.
type OpenAIClient struct {
	client openai.Client
}

// NewOpenAIClient returns a client. baseURL defaults to http://localhost:12434/engines/v1 if empty.
// When apiKey is empty, Authorization is stripped on each request so local servers work even if
// OPENAI_API_KEY is set in the environment.
// httpClient defaults to the SDK default when nil.
func NewOpenAIClient(baseURL, apiKey string, httpClient *http.Client) *OpenAIClient {
	b := strings.TrimSpace(baseURL)
	if b == "" {
		b = defaultOpenAIBase
	}

	opts := []option.RequestOption{
		option.WithBaseURL(b),
	}
	key := strings.TrimSpace(apiKey)
	if key != "" {
		opts = append(opts, option.WithAPIKey(key))
	} else {
		opts = append(opts, option.WithMiddleware(stripAuthorizationMiddleware))
	}
	if httpClient != nil {
		opts = append(opts, option.WithHTTPClient(httpClient))
	}

	return &OpenAIClient{client: openai.NewClient(opts...)}
}

func stripAuthorizationMiddleware(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
	req.Header.Del("Authorization")
	return next(req)
}

// Complete sends messages to the upstream API and returns the assistant text.
func (c *OpenAIClient) Complete(ctx context.Context, model string, messages []apiChatMessage) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("no messages")
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(model),
		Messages: toSDKMessages(messages),
	}

	completion, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", mapChatError(err)
	}
	if len(completion.Choices) == 0 || strings.TrimSpace(completion.Choices[0].Message.Content) == "" {
		return "", &upstreamError{status: http.StatusBadGateway, msg: "empty model response"}
	}
	return completion.Choices[0].Message.Content, nil
}

func toSDKMessages(messages []apiChatMessage) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "system":
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(m.Content),
					},
				},
			})
		case "user":
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(m.Content),
					},
				},
			})
		case "assistant":
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(m.Content),
					},
				},
			})
		}
	}
	return out
}

func mapChatError(err error) error {
	var aerr *openai.Error
	if errors.As(err, &aerr) {
		msg := strings.TrimSpace(aerr.Message)
		if msg == "" {
			msg = "request failed"
		}
		return &upstreamError{status: mapUpstreamStatus(aerr.StatusCode), msg: msg}
	}
	// Network or other non-API failures
	text := strings.TrimSpace(err.Error())
	if len(text) > 200 {
		text = text[:200] + "…"
	}
	if text == "" {
		text = "could not reach model server"
	}
	return &upstreamError{status: http.StatusBadGateway, msg: text}
}

func mapUpstreamStatus(code int) int {
	switch code {
	case http.StatusUnauthorized:
		return http.StatusUnauthorized
	case http.StatusTooManyRequests:
		return http.StatusTooManyRequests
	case http.StatusBadRequest:
		return http.StatusBadGateway
	default:
		if code >= 500 {
			return http.StatusBadGateway
		}
		if code >= 400 {
			return http.StatusBadGateway
		}
		return http.StatusBadGateway
	}
}

type upstreamError struct {
	status int
	msg    string
}

func (e *upstreamError) Error() string {
	return e.msg
}
