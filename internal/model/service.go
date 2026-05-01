package model

import (
	"errors"
	"sync"
)

// ErrUnknownModel is returned when the id is not in the catalog.
var ErrUnknownModel = errors.New("unknown model id")

// Service holds the process-wide selected model (replace with per-session storage later).
type Service struct {
	mu         sync.Mutex
	selectedID string
}

// NewService returns a service with selection initialized to the first catalog entry.
func NewService() *Service {
	first := ""
	if len(Catalog) > 0 {
		first = Catalog[0].ID
	}
	return &Service{selectedID: first}
}

// List returns a copy of the catalog and the current selected id.
func (s *Service) List() ([]Model, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Model, len(Catalog))
	copy(out, Catalog)
	return out, s.selectedID
}

// Select validates id, stores it, and returns the model.
func (s *Service) Select(id string) (Model, error) {
	m, ok := byID(id)
	if !ok {
		return Model{}, ErrUnknownModel
	}
	s.mu.Lock()
	s.selectedID = id
	s.mu.Unlock()
	return m, nil
}

// Selected returns the currently selected model from the catalog.
func (s *Service) Selected() (Model, error) {
	s.mu.Lock()
	id := s.selectedID
	s.mu.Unlock()
	m, ok := byID(id)
	if !ok {
		return Model{}, ErrUnknownModel
	}
	return m, nil
}
