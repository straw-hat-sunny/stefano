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
	t.Setenv("TAVILY_API_KEY", "")

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
	if _, has := gotBody["tools"]; has {
		t.Fatalf("expected no tools in request when TAVILY_API_KEY is unset")
	}
}

func TestOpenAILLMClient_requestIncludesWebSearchToolWhenTavilyConfigured(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
						"content": "ok",
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
	t.Setenv("TAVILY_API_KEY", "tavily-test-key")

	c, err := NewOpenAILLMClient()
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.GenerateMessage(context.Background(), "hi")
	if err != nil {
		t.Fatal(err)
	}
	tools, ok := gotBody["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("expected non-empty tools in request body, got %#v", gotBody["tools"])
	}
	t0, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("tool entry type: %T", tools[0])
	}
	fn, ok := t0["function"].(map[string]any)
	if !ok {
		t.Fatalf("tool.function type: %T", t0["function"])
	}
	if fn["name"] != webSearchToolName {
		t.Fatalf("tool name: got %v want %q", fn["name"], webSearchToolName)
	}
}

func TestNewOpenAILLMClient_invalidBaseURL(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "://")
	_, err := NewOpenAILLMClient()
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}
