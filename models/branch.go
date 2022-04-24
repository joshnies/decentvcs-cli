package models

type Branch struct {
	ID        string `json:"_id,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	Name      string `json:"name,omitempty"`
	CommitID  string `json:"commit_id,omitempty"`
}
