package models

type FileChangeType int

const (
	FileWasCreated FileChangeType = iota
	FileWasModified
	FileWasDeleted
)

type FileChange struct {
	Path string         `json:"path,omitempty"`
	Type FileChangeType `json:"type,omitempty"`
}
