package models

type Project struct {
	ID                    string   `json:"_id,omitempty"`
	CreatedAt             int64    `json:"created_at,omitempty"`
	Name                  string   `json:"name,omitempty"`
	Branches              []Branch `json:"branches,omitempty"`
	AccessGrant           string   `json:"access_grant,omitempty"`
	AccessGrantExpiration int64    `json:"access_grant_expiration,omitempty"`
}

type ProjectWithBranchesAndCommit struct {
	ID                    string             `json:"_id,omitempty"`
	CreatedAt             int64              `json:"created_at,omitempty"`
	Name                  string             `json:"name,omitempty"`
	Branches              []BranchWithCommit `json:"branches,omitempty"`
	AccessGrant           string             `json:"access_grant,omitempty"`
	AccessGrantExpiration int64              `json:"access_grant_expiration,omitempty"`
}
