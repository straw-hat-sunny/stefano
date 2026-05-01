package model

// Catalog is the supported model list (hardcoded until backed by config or DB).
var Catalog = []Model{
	{ID: "gpt-4.1", Label: "GPT-4.1"},
	{ID: "gpt-4o-mini", Label: "GPT-4o mini"},
	{ID: "claude-sonnet", Label: "Claude Sonnet"},
	{ID: "gemma4", Label: "Gemma 4"},
}

// Model is a selectable chat model.
type Model struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func byID(id string) (Model, bool) {
	for _, m := range Catalog {
		if m.ID == id {
			return m, true
		}
	}
	return Model{}, false
}

// Lookup returns the catalog entry for id, if any.
func Lookup(id string) (Model, bool) {
	return byID(id)
}
