package models

// Data for a file in a commit.
type FileData struct {
	// File hash
	Hash string `json:"hash"`
	// Patch files hashes
	PatchHashes []string `json:"patch_hashes"`
	// Version number of this file. Starts at 1.
	// For example, if the file has been uploaded and changed twice, then this will be 3.
	Version uint8 `json:"version"`
}
