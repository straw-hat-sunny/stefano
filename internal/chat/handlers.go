package chat

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	maxBodyBytes = 1 << 16
	replyDelay   = 800 * time.Millisecond
)

// replySleeper is overridden in tests to avoid real delays.
var replySleeper = time.Sleep

type messageRequest struct {
	Content string `json:"content"`
	ModelID string `json:"modelId"`
}

type messagePayload struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageResponse struct {
	Message messagePayload `json:"message"`
}

type errResponse struct {
	Error string `json:"error"`
}

// HandleMessage serves POST /api/chat/message with JSON body {"content":"...","modelId":"..."}.
func HandleMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body) > maxBodyBytes {
		writeErr(w, http.StatusBadRequest, "body too large")
		return
	}

	var req messageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		writeErr(w, http.StatusBadRequest, "content is required")
		return
	}

	replySleeper(replyDelay)

	modelRef := strings.TrimSpace(req.ModelID)
	if modelRef == "" {
		modelRef = "(no model)"
	}
	text := fmt.Sprintf(
		"(%s) Hi John!.",
		modelRef,
	)

	msgID, err := randomHexID()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	_ = json.NewEncoder(w).Encode(messageResponse{
		Message: messagePayload{
			ID:      msgID,
			Role:    "assistant",
			Content: text,
		},
	})
}

func randomHexID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errResponse{Error: msg})
}
