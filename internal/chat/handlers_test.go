package chat

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	replySleeper = func(time.Duration) {}
	os.Exit(m.Run())
}

func TestHandleMessage_badJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/api/chat/message", bytes.NewReader([]byte(`{`))))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestHandleMessage_emptyContent(t *testing.T) {
	rec := httptest.NewRecorder()
	body := []byte(`{"content":"   ","modelId":"x"}`)
	HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/api/chat/message", bytes.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestHandleMessage_ok(t *testing.T) {
	rec := httptest.NewRecorder()
	body := []byte(`{"content":"hello","modelId":"gpt-demo"}`)
	HandleMessage(rec, httptest.NewRequest(http.MethodPost, "/api/chat/message", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	var got messageResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Message.Role != "assistant" || got.Message.ID == "" || got.Message.Content == "" {
		t.Fatalf("unexpected %+v", got.Message)
	}
}
