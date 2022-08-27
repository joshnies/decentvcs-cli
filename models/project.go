package models

import "time"

type Project struct {
	ID                   string    `json:"_id,omitempty"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
	Name                 string    `json:"name,omitempty"`
	TeamID               string    `json:"team_id,omitempty"`
	Branches             []Branch  `json:"branches,omitempty"`
	DefaultBranchID      string    `json:"default_branch_id,omitempty"`
	EnablePatchRevisions bool      `json:"enable_patch_revisions,omitempty"`
}

type ProjectWithBranchesAndCommit struct {
	ID                   string             `json:"_id,omitempty"`
	CreatedAt            time.Time          `json:"created_at,omitempty"`
	Name                 string             `json:"name,omitempty"`
	TeamID               string             `json:"team_id,omitempty"`
	Branches             []BranchWithCommit `json:"branches,omitempty"`
	DefaultBranchID      string             `json:"default_branch_id,omitempty"`
	EnablePatchRevisions bool               `json:"enable_patch_revisions,omitempty"`
}

type CreateProjectRequest struct {
	// URL of the thumbnail image.
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	// If `true`, modified committed files in this project will be uploaded as patches instead of snapshots (e.g. the
	// whole file).
	EnablePatchRevisions bool `json:"enable_patch_revisions,omitempty"`
}
