package chat

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-assistant/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	if status != 0 {
		w.WriteHeader(status)
	}
	_, _ = w.Write(body)
}

func TestHandleMessage_badJSON(t *testing.T) {
	h := newTestHandler(t, nil)
	rec := httptest.NewRecorder()
	h.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/api/chat/message", bytes.NewReader([]byte(`{`))))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestHandleMessage_emptyContentNoMessages(t *testing.T) {
	h := newTestHandler(t, nil)
	rec := httptest.NewRecorder()
	body := []byte(`{"content":"   ","modelId":"gpt-4.1"}`)
	h.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/api/chat/message", bytes.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestHandleMessage_unknownModel(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream should not be called")
	}))
	defer up.Close()

	h := newTestHandlerWithURL(t, up.URL+"/v1", up.Client())
	rec := httptest.NewRecorder()
	body := []byte(`{"content":"hello","modelId":"not-in-catalog"}`)
	h.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/api/chat/message", bytes.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestHandleMessage_emptyAPIKeyNoAuthHeader(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Fatal("expected no Authorization header when api key is empty")
		}
		writeJSON(w, http.StatusOK, []byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer up.Close()

	models := model.NewService()
	h := NewHandler(models, NewOpenAIClient(up.URL+"/v1", "", up.Client()))
	rec := httptest.NewRecorder()
	body := []byte(`{"content":"hello","modelId":"gpt-4.1"}`)
	h.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/api/chat/message", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestHandleMessage_ok(t *testing.T) {
	var gotModel string
	var gotMsgs []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path %s", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(b, &req); err != nil {
			t.Fatal(err)
		}
		gotModel = req.Model
		gotMsgs = req.Messages
		writeJSON(w, http.StatusOK, []byte(`{"choices":[{"message":{"role":"assistant","content":"hello back"}}]}`))
	}))
	defer up.Close()

	h := newTestHandlerWithURL(t, up.URL+"/v1", up.Client())
	rec := httptest.NewRecorder()
	body := []byte(`{"content":"hello","modelId":"gpt-4o-mini","messages":[{"role":"system","content":"You are Stefano."},{"role":"user","content":"hello"}]}`)
	h.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/api/chat/message", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d body %s", rec.Code, rec.Body.String())
	}
	if gotModel != "gpt-4o-mini" {
		t.Fatalf("model %q", gotModel)
	}
	if len(gotMsgs) != 2 {
		t.Fatalf("want 2 messages (system + user), got %+v", gotMsgs)
	}
	if gotMsgs[0].Role != "system" || gotMsgs[0].Content != "You are Stefano." {
		t.Fatalf("first message %+v", gotMsgs[0])
	}
	if gotMsgs[1].Role != "user" || gotMsgs[1].Content != "hello" {
		t.Fatalf("second message %+v", gotMsgs[1])
	}
	var got messageResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Message.Role != "assistant" || got.Message.ID == "" || got.Message.Content != "hello back" {
		t.Fatalf("unexpected %+v", got.Message)
	}
}

func TestHandleMessage_upstreamError(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusUnauthorized, []byte(`{"error":{"message":"bad key","type":"invalid_request_error","code":"","param":""}}`))
	}))
	defer up.Close()

	h := newTestHandlerWithURL(t, up.URL+"/v1", up.Client())
	rec := httptest.NewRecorder()
	body := []byte(`{"content":"x","modelId":"gpt-4.1"}`)
	h.HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/api/chat/message", bytes.NewReader(body)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestBuildAPIMessages_invalidRole(t *testing.T) {
	_, err := buildAPIMessages(&messageRequest{
		Messages: []incomingMessage{{Role: "developer", Content: "x"}},
	})
	if err == nil || !strings.Contains(err.Error(), "system, user, or assistant") {
		t.Fatalf("got %v", err)
	}
}

func newTestHandler(t *testing.T, client *http.Client) *Handler {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, []byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	t.Cleanup(srv.Close)
	return newTestHandlerWithURL(t, srv.URL+"/v1", clientOrDefault(srv, client))
}

func newTestHandlerWithURL(t *testing.T, baseURL string, client *http.Client) *Handler {
	t.Helper()
	if client == nil {
		client = http.DefaultClient
	}
	return NewHandler(model.NewService(), NewOpenAIClient(baseURL, "test-key", client))
}

func clientOrDefault(srv *httptest.Server, c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return srv.Client()
}
