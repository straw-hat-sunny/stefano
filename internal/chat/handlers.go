package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"ai-assistant/internal/model"
)

const (
	maxBodyBytes         = 1 << 16
	maxChatMessages      = 100
	maxMessageRunes      = 32000
	maxTotalPayloadRunes = 120000
)

// apiChatMessage is one turn passed to the completion backend.
type apiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionClient interface {
	Complete(ctx context.Context, model string, messages []apiChatMessage) (string, error)
}

// Handler serves chat HTTP endpoints.
type Handler struct {
	Models *model.Service
	OpenAI completionClient
}

// NewHandler returns a Handler. openAI may be nil only in tests that replace behavior.
func NewHandler(models *model.Service, openAI completionClient) *Handler {
	return &Handler{Models: models, OpenAI: openAI}
}

type incomingMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageRequest struct {
	Content  string            `json:"content"`
	ModelID  string            `json:"modelId"`
	Messages []incomingMessage `json:"messages"`
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

// HandleMessage serves POST /api/chat/message with JSON body:
// {"content":"...","modelId":"...","messages":[{"role":"system|user|assistant","content":"..."}]}.
// The client should send the full transcript (typically system first, then alternating user/assistant).
// When messages is non-empty it is sent to the model (capped); otherwise content is required as a single user turn.
func (h *Handler) HandleMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.Models == nil || h.OpenAI == nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

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

	apiMsgs, err := buildAPIMessages(&req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	modelID, err := h.resolveModelID(strings.TrimSpace(req.ModelID))
	if err != nil {
		if errors.Is(err, model.ErrUnknownModel) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	ctx := r.Context()
	text, err := h.OpenAI.Complete(ctx, modelID, apiMsgs)
	if err != nil {
		var ue *upstreamError
		if errors.As(err, &ue) {
			writeErr(w, ue.status, ue.msg)
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

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

func (h *Handler) resolveModelID(fromBody string) (string, error) {
	if fromBody != "" {
		if _, ok := model.Lookup(fromBody); !ok {
			return "", model.ErrUnknownModel
		}
		return fromBody, nil
	}
	m, err := h.Models.Selected()
	if err != nil {
		return "", err
	}
	return m.ID, nil
}

func buildAPIMessages(req *messageRequest) ([]apiChatMessage, error) {
	var raw []incomingMessage
	if len(req.Messages) > 0 {
		raw = req.Messages
	} else {
		content := strings.TrimSpace(req.Content)
		if content == "" {
			return nil, errors.New("content is required")
		}
		raw = []incomingMessage{{Role: "user", Content: content}}
	}

	if len(raw) > maxChatMessages {
		if len(raw) > 0 && strings.ToLower(strings.TrimSpace(raw[0].Role)) == "system" {
			rest := raw[1:]
			if len(rest) > maxChatMessages-1 {
				rest = rest[len(rest)-(maxChatMessages-1):]
			}
			raw = append([]incomingMessage{raw[0]}, rest...)
		} else {
			raw = raw[len(raw)-maxChatMessages:]
		}
	}

	var out []apiChatMessage
	for _, m := range raw {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role != "system" && role != "user" && role != "assistant" {
			return nil, errors.New("messages must use role system, user, or assistant")
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			return nil, errors.New("message content must be non-empty")
		}
		runes := []rune(content)
		if len(runes) > maxMessageRunes {
			content = string(runes[len(runes)-maxMessageRunes:])
		}
		out = append(out, apiChatMessage{Role: role, Content: content})
	}

	for countRunesMessages(out) > maxTotalPayloadRunes && len(out) > 1 {
		if len(out) > 0 && out[0].Role == "system" {
			out = append([]apiChatMessage{out[0]}, out[2:]...)
			if len(out) == 1 {
				break
			}
			continue
		}
		out = out[1:]
	}

	return out, nil
}

func countRunesMessages(msgs []apiChatMessage) int {
	n := 0
	for _, m := range msgs {
		n += len([]rune(m.Content))
	}
	return n
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
