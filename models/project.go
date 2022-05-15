package models

type Project struct {
	ID                        string   `json:"_id,omitempty"`
	CreatedAt                 int64    `json:"created_at,omitempty"`
	OwnerID                   string   `json:"owner_id,omitempty"`
	Name                      string   `json:"name,omitempty"`
	Branches                  []Branch `json:"branches,omitempty"`
	StorjAccessGrant          string   `json:"storj_access_grant,omitempty"`
	StorjAccessGrantExpiresAt int64    `json:"storj_access_grant_expires_at,omitempty"`
	DefaultBranchID           string   `json:"default_branch_id,omitempty"`
}

type ProjectWithBranchesAndCommit struct {
	ID                        string             `json:"_id,omitempty"`
	CreatedAt                 int64              `json:"created_at,omitempty"`
	OwnerID                   string             `json:"owner_id,omitempty"`
	Name                      string             `json:"name,omitempty"`
	Branches                  []BranchWithCommit `json:"branches,omitempty"`
	StorjAccessGrant          string             `json:"storj_access_grant,omitempty"`
	StorjAccessGrantExpiresAt int64              `json:"storj_access_grant_expires_at,omitempty"`
	DefaultBranchID           string             `json:"default_branch_id,omitempty"`
}
