package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// chatTestsMockServer serves OpenAI-compatible chat completions so CreateChat's embedded client matches ProcessMessage expectations.
var chatTestsMockServer *httptest.Server

func TestMain(m *testing.M) {
	chatTestsMockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
						"content": "Thinking...",
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
	defer chatTestsMockServer.Close()

	_ = os.Setenv("OPENAI_BASE_URL", chatTestsMockServer.URL+"/engines/v1")
	_ = os.Setenv("OPENAI_MODEL", "gemma4")
	_ = os.Setenv("OPENAI_API_KEY", "test")

	os.Exit(m.Run())
}

func setupTestRouter() (*mux.Router, *Service) {
	svc := NewService(NewInMemRepo())
	r := mux.NewRouter()
	RegisterRoutes(r, svc)
	return r, svc
}

func TestHandleCreateChat_OK(t *testing.T) {
	r, _ := setupTestRouter()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", nil)

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type: got %q want application/json", ct)
	}

	var got ChatResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID == uuid.Nil {
		t.Fatal("expected non-nil chat id")
	}
	if len(got.Messages) != 1 {
		t.Fatalf("messages len: got %d want 1", len(got.Messages))
	}
	if got.Messages[0].User != assistantUser {
		t.Fatalf("first message user: got %q want %q", got.Messages[0].User, assistantUser)
	}
	if got.Messages[0].Content != assistantMessage {
		t.Fatalf("first message content mismatch")
	}
}

func TestHandleCreateChat_RepoError(t *testing.T) {
	svc := NewService(&createFailRepo{})
	r := mux.NewRouter()
	RegisterRoutes(r, svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusInternalServerError)
	}
	var er errResponse
	if err := json.NewDecoder(rec.Body).Decode(&er); err != nil {
		t.Fatal(err)
	}
	if er.Error != "internal error" {
		t.Fatalf("error message: got %q", er.Error)
	}
}

type createFailRepo struct{}

func (createFailRepo) CreateChat(context.Context, Chat) error { return errors.New("fail") }
func (createFailRepo) GetChat(context.Context, uuid.UUID) (Chat, error) {
	return Chat{}, ErrChatNotFound
}
func (createFailRepo) UpdateChat(context.Context, Chat) error      { return nil }
func (createFailRepo) DeleteChat(context.Context, uuid.UUID) error { return nil }

func TestHandleGetChat_OK(t *testing.T) {
	r, _ := setupTestRouter()

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/chat", nil)
	r.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status: %d", createRec.Code)
	}
	var created ChatResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chat/"+created.ID.String(), nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d body=%s", rec.Code, rec.Body.String())
	}
	var got ChatResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID {
		t.Fatalf("id: got %v want %v", got.ID, created.ID)
	}
	if len(got.Messages) != len(created.Messages) {
		t.Fatalf("messages len mismatch")
	}
}

func TestHandleGetChat_NotFound(t *testing.T) {
	r, _ := setupTestRouter()
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chat/"+id.String(), nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want %d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	var er errResponse
	if err := json.NewDecoder(rec.Body).Decode(&er); err != nil {
		t.Fatal(err)
	}
	if er.Error != "chat not found" {
		t.Fatalf("error: got %q", er.Error)
	}
}

func TestHandleGetChat_MalformedChatID_NoRouteMatch(t *testing.T) {
	// mux UUID regex rejects malformed ids — no handler runs.
	r, _ := setupTestRouter()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chat/not-a-uuid", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404 for non-matching route", rec.Code)
	}
}

func TestHandleProcessChat_OK(t *testing.T) {
	r, _ := setupTestRouter()

	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/chat", nil))
	var created ChatResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	body := `{"content":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat/"+created.ID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d body=%s", rec.Code, rec.Body.String())
	}
	var got ProcessChatResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Message.User != assistantUser {
		t.Fatalf("assistant user: got %q", got.Message.User)
	}
	if got.Message.Content != "Thinking..." {
		t.Fatalf("assistant content: got %q", got.Message.Content)
	}
}

func TestHandleProcessChat_NotFound(t *testing.T) {
	r, _ := setupTestRouter()
	id := uuid.MustParse("00000000-0000-4000-8000-000000000002")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat/"+id.String(), strings.NewReader(`{"content":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d", rec.Code)
	}
}

func TestHandleProcessChat_BadRequest(t *testing.T) {
	r, _ := setupTestRouter()

	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/chat", nil))
	var created ChatResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	base := "/api/chat/" + created.ID.String()

	tests := []struct {
		name       string
		body       io.Reader
		wantSubstr string
	}{
		{
			name:       "empty content",
			body:       strings.NewReader(`{"content":""}`),
			wantSubstr: "expected JSON object with non-empty content",
		},
		{
			name:       "invalid json",
			body:       strings.NewReader(`not json`),
			wantSubstr: "expected JSON object with non-empty content",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, base, tt.body)
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status: got %d body=%s", rec.Code, rec.Body.String())
			}
			var er errResponse
			if err := json.NewDecoder(rec.Body).Decode(&er); err != nil {
				t.Fatal(err)
			}
			if er.Error != tt.wantSubstr {
				t.Fatalf("error: got %q want %q", er.Error, tt.wantSubstr)
			}
		})
	}
}

func TestHandleProcessChat_BodyTooLarge(t *testing.T) {
	r, _ := setupTestRouter()

	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/chat", nil))
	var created ChatResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	// total length > maxChatBodyBytes (LimitReader max is maxChatBodyBytes+1)
	large := `{"content":"` + strings.Repeat("x", maxChatBodyBytes-12) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat/"+created.ID.String(), strings.NewReader(large))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d", rec.Code)
	}
	var er errResponse
	if err := json.NewDecoder(rec.Body).Decode(&er); err != nil {
		t.Fatal(err)
	}
	if er.Error != "body too large" {
		t.Fatalf("error: got %q", er.Error)
	}
}

func TestToChatResponse(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	c := Chat{
		ID: id,
		Messages: []Message{
			{User: "user", Content: "a"},
			{User: assistantUser, Content: "b"},
		},
	}
	got := toChatResponse(c)
	if got.ID != id {
		t.Fatal("id")
	}
	if len(got.Messages) != 2 || got.Messages[0].Content != "a" || got.Messages[1].Content != "b" {
		t.Fatalf("messages: %+v", got.Messages)
	}
}

func TestWriteErr(t *testing.T) {
	rec := httptest.NewRecorder()
	writeErr(rec, http.StatusTeapot, "x")
	if rec.Code != http.StatusTeapot {
		t.Fatalf("code: %d", rec.Code)
	}
	var er errResponse
	if err := json.NewDecoder(rec.Body).Decode(&er); err != nil {
		t.Fatal(err)
	}
	if er.Error != "x" {
		t.Fatal(er.Error)
	}
}

func TestHandleProcessChat_ReadBodyError(t *testing.T) {
	r, _ := setupTestRouter()

	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/chat", nil))
	var created ChatResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat/"+created.ID.String(), errReader{})
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d body=%s", rec.Code, rec.Body.String())
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read error") }
