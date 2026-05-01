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
	m, err := s.Select("gpt-4o-mini")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "gpt-4o-mini" || m.Label != "GPT-4o mini" {
		t.Fatalf("model: %+v", m)
	}
	_, sel := s.List()
	if sel != "gpt-4o-mini" {
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

func TestService_Selected(t *testing.T) {
	s := NewService()
	m, err := s.Selected()
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != Catalog[0].ID {
		t.Fatalf("got %q want %q", m.ID, Catalog[0].ID)
	}
	if _, err := s.Select("gpt-4o-mini"); err != nil {
		t.Fatal(err)
	}
	m2, err := s.Selected()
	if err != nil {
		t.Fatal(err)
	}
	if m2.ID != "gpt-4o-mini" {
		t.Fatalf("got %q", m2.ID)
	}
}
