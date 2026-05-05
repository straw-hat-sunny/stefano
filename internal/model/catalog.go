package model

// Catalog is the supported model list (hardcoded until backed by config or DB).
var Catalog = []Model{
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
