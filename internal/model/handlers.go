package model

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const maxSelectBodyBytes = 1 << 16

type listResponse struct {
	Models     []Model `json:"models"`
	SelectedID string  `json:"selectedId"`
}

type selectRequest struct {
	ID string `json:"id"`
}

type errResponse struct {
	Error string `json:"error"`
}

// HandleList serves GET /api/models.
func (s *Service) HandleList(w http.ResponseWriter, _ *http.Request) {
	models, selectedID := s.List()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(listResponse{Models: models, SelectedID: selectedID})
}

// HandleSelect serves POST /api/model with JSON body {"id":"..."}.
func (s *Service) HandleSelect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(io.LimitReader(r.Body, maxSelectBodyBytes+1))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body) > maxSelectBodyBytes {
		writeErr(w, http.StatusBadRequest, "body too large")
		return
	}

	var req selectRequest
	if err := json.Unmarshal(body, &req); err != nil || req.ID == "" {
		writeErr(w, http.StatusBadRequest, "expected JSON object with non-empty id")
		return
	}

	m, err := s.Select(req.ID)
	if err != nil {
		if errors.Is(err, ErrUnknownModel) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	_ = json.NewEncoder(w).Encode(m)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errResponse{Error: msg})
}
