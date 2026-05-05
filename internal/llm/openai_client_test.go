package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAILLMClient_GenerateMessage_httptest(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-test",
			"object": "chat.completion",
			"created": 1700000000,
			"model": "gemma4",
			"choices": [
				{
					"index": 0,
					"finish_reason": "stop",
					"message": {
						"role": "assistant",
						"content": "mock reply",
						"refusal": ""
					},
					"logprobs": {
						"content": [],
						"refusal": []
					}
				}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("OPENAI_BASE_URL", srv.URL+"/engines/v1")
	t.Setenv("OPENAI_MODEL", "gemma4")
	t.Setenv("OPENAI_API_KEY", "test-key")

	c, err := NewOpenAILLMClient()
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.GenerateMessage(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if out != "mock reply" {
		t.Fatalf("content: got %q want %q", out, "mock reply")
	}
	if !strings.HasSuffix(gotPath, "/chat/completions") {
		t.Fatalf("path: got %q want suffix /chat/completions", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Fatalf("Authorization: got %q want Bearer prefix", gotAuth)
	}
	if m, _ := gotBody["model"].(string); m != "gemma4" {
		t.Fatalf("request model: got %v want gemma4", gotBody["model"])
	}
}

func TestNewOpenAILLMClient_invalidBaseURL(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "://")
	_, err := NewOpenAILLMClient()
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}
