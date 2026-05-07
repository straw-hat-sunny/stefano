package chat

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const maxChatBodyBytes = 1 << 16

// chatIDPath is a mux path segment with UUID constraint for {chat_id}.
const chatIDPath = `/{chat_id:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}}`

type chatMessageDTO struct {
	ID      uuid.UUID `json:"id"`
	User    string    `json:"user"`
	Content string    `json:"content"`
}

// ChatResponse is the JSON shape for a chat session.
type ChatResponse struct {
	ID       uuid.UUID        `json:"id"`
	Messages []chatMessageDTO `json:"messages"`
}

// ProcessChatRequest is the body for POST /api/chat/{chat_id}.
type ProcessChatRequest struct {
	Content string `json:"content"`
}

// ProcessChatResponse is the response for POST /api/chat/{chat_id}.
type ProcessChatResponse struct {
	Message chatMessageDTO `json:"message"`
}

type errResponse struct {
	Error string `json:"error"`
}

// RegisterRoutes mounts all /api/chat HTTP routes on parent.
func RegisterRoutes(parent *mux.Router, svc *Service) {
	parent.Methods(http.MethodPost).Path("/api/chat").HandlerFunc(svc.HandleCreateChat)

	sr := parent.PathPrefix("/api/chat").Subrouter()
	sr.Methods(http.MethodGet).Path(chatIDPath).HandlerFunc(svc.HandleGetChat)
	sr.Methods(http.MethodPost).Path(chatIDPath).HandlerFunc(svc.HandleProcessChat)
}

// HandleCreateChat serves POST /api/chat.
func (s *Service) HandleCreateChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	chat, err := s.CreateChat(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	_ = json.NewEncoder(w).Encode(toChatResponse(chat))
}

// HandleGetChat serves GET /api/chat/{chat_id}.
func (s *Service) HandleGetChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, err := uuid.Parse(mux.Vars(r)["chat_id"])
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid chat id")
		return
	}
	chat, err := s.GetChat(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrChatNotFound) {
			writeErr(w, http.StatusNotFound, "chat not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	_ = json.NewEncoder(w).Encode(toChatResponse(chat))
}

// HandleProcessChat serves POST /api/chat/{chat_id}.
func (s *Service) HandleProcessChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, err := uuid.Parse(mux.Vars(r)["chat_id"])
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid chat id")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxChatBodyBytes+1))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body) > maxChatBodyBytes {
		writeErr(w, http.StatusBadRequest, "body too large")
		return
	}

	var req ProcessChatRequest
	if err := json.Unmarshal(body, &req); err != nil || req.Content == "" {
		writeErr(w, http.StatusBadRequest, "expected JSON object with non-empty content")
		return
	}

	userMsg, err := NewMessage("user", req.Content)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	assistantMsg, err := s.ProcessMessage(r.Context(), id, userMsg)
	if err != nil {
		if errors.Is(err, ErrChatNotFound) {
			writeErr(w, http.StatusNotFound, "chat not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	_ = json.NewEncoder(w).Encode(ProcessChatResponse{
		Message: messageToDTO(assistantMsg),
	})
}

func messageToDTO(m Message) chatMessageDTO {
	return chatMessageDTO{ID: m.ID, User: m.User, Content: m.Content}
}

func toChatResponse(c Chat) ChatResponse {
	dtos := make([]chatMessageDTO, len(c.Messages))
	for i := range c.Messages {
		dtos[i] = messageToDTO(c.Messages[i])
	}
	return ChatResponse{ID: c.ID, Messages: dtos}
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errResponse{Error: msg})
}
