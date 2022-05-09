package models

type FileChangeType int

const (
	FileWasCreated FileChangeType = iota
	FileWasModified
	FileWasDeleted
)
