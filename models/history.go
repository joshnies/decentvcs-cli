package models

// Project history object.
// Multiple of these are saved in a JSON array in the project history file.
type HistoryEntry struct {
	Path string `json:"path,omitempty"`
	Hash string `json:"hash,omitempty"`
}
