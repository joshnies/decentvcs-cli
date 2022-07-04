package models

type Project struct {
	ID              string   `json:"_id,omitempty"`
	CreatedAt       int64    `json:"created_at,omitempty"`
	Name            string   `json:"name,omitempty"`
	Blob            string   `json:"blob,omitempty"`
	TeamID          string   `json:"team_id,omitempty"`
	Branches        []Branch `json:"branches,omitempty"`
	DefaultBranchID string   `json:"default_branch_id,omitempty"`
}

type ProjectWithBranchesAndCommit struct {
	ID              string             `json:"_id,omitempty"`
	CreatedAt       int64              `json:"created_at,omitempty"`
	Name            string             `json:"name,omitempty"`
	Blob            string             `json:"blob,omitempty"`
	TeamID          string             `json:"team_id,omitempty"`
	Branches        []BranchWithCommit `json:"branches,omitempty"`
	DefaultBranchID string             `json:"default_branch_id,omitempty"`
}
