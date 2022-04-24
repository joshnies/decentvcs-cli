package models

type Project struct {
	ID        string   `json:"_id,omitempty"`
	CreatedAt int64    `json:"created_at,omitempty"`
	Name      string   `json:"name,omitempty"`
	Branches  []Branch `json:"branches,omitempty"`
}
