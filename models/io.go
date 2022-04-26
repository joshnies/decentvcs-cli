package models

type ProjectFileData struct {
	ProjectID             string `json:"project_id,omitempty"`
	CurrentBranchID       string `json:"current_branch_id,omitempty"`
	AccessGrant           string `json:"access_grant,omitempty"`
	AccessGrantExpiration int64  `json:"access_grant_expiration,omitempty"`
}

func MergeProjectFileData(existingData, newData ProjectFileData) ProjectFileData {
	merged := existingData

	if newData.ProjectID != "" {
		merged.ProjectID = newData.ProjectID
	}

	if newData.CurrentBranchID != "" {
		merged.CurrentBranchID = newData.CurrentBranchID
	}

	if newData.AccessGrant != "" {
		merged.AccessGrant = newData.AccessGrant
	}

	if newData.AccessGrantExpiration != 0 {
		merged.AccessGrantExpiration = newData.AccessGrantExpiration
	}

	return merged
}
