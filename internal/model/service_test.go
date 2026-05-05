package model

import (
	"errors"
	"testing"
)

func TestNewService_DefaultSelection(t *testing.T) {
	s := NewService()
	models, sel := s.List()
	if len(models) != len(Catalog) {
		t.Fatalf("list length: got %d want %d", len(models), len(Catalog))
	}
	if sel != Catalog[0].ID {
		t.Fatalf("selectedId: got %q want %q", sel, Catalog[0].ID)
	}
}

func TestService_Select(t *testing.T) {
	s := NewService()
	m, err := s.Select("gemma4")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "gemma4" || m.Label != "Gemma 4" {
		t.Fatalf("model: %+v", m)
	}
	_, sel := s.List()
	if sel != "gemma4" {
		t.Fatalf("selectedId after select: got %q", sel)
	}
}

func TestService_SelectUnknown(t *testing.T) {
	s := NewService()
	_, err := s.Select("nope")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnknownModel) {
		t.Fatalf("want ErrUnknownModel, got %v", err)
	}
}
