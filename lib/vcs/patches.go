package vcs

import (
	"os"

	"github.com/gabstv/go-bsdiff/pkg/bsdiff"
	"github.com/gabstv/go-bsdiff/pkg/bspatch"
)

// Generate a patch between two files.
// Uses bsdiff.
func GenPatch(fromPath string, toPath string) ([]byte, error) {
	// Read "from" file
	fromFileBytes, err := os.ReadFile(fromPath)
	if err != nil {
		return nil, err
	}

	// Read "to" file
	toFileBytes, err := os.ReadFile(toPath)
	if err != nil {
		return nil, err
	}

	// Create and return patchBytes as bytes
	patchBytes, err := bsdiff.Bytes(fromFileBytes, toFileBytes)
	if err != nil {
		return nil, err
	}

	return patchBytes, nil
}

// Generate a patch between two files and write the patch to a file.
// Uses bsdiff.
func GenPatchFile(fromPath string, toPath string, patchPath string) error {
	patchBytes, err := GenPatch(fromPath, toPath)
	if err != nil {
		return err
	}

	// Write patchBytes to patchPath
	err = os.WriteFile(patchPath, patchBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Apply a patch to a file.
// Uses bspatch.
func ApplyPatch(oldFilePath string, patchFilePath string) ([]byte, error) {
	// Read old file
	oldFileBytes, err := os.ReadFile(oldFilePath)
	if err != nil {
		return nil, err
	}

	// Read patch files
	patchFileBytes, err := os.ReadFile(patchFilePath)
	if err != nil {
		return nil, err
	}

	// Create and return result as bytes
	result, err := bspatch.Bytes(oldFileBytes, patchFileBytes)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Apply a patch to a file and write the result to a file.
// Uses bspatch.
func ApplyPatchAsNewFile(oldFilePath string, patchFilePath string, newFilePath string) error {
	result, err := ApplyPatch(oldFilePath, patchFilePath)
	if err != nil {
		return err
	}

	// Write result to newFilePath
	err = os.WriteFile(newFilePath, result, 0644)
	if err != nil {
		return err
	}

	return nil
}
