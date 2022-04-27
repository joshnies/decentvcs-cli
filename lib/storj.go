package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/models"
	"storj.io/uplink"
)

// Get Storj access grant for project.
// Gives read-write access to project folder in Storj bucket.
//
// Returns Storj access grant (unserialized).
func GetAccessGrant() (*uplink.Access, error) {
	// Use existing access grant if it exists and has not expired
	projectConfig, err := config.GetProjectConfig()
	if err == nil {
		expiration := time.Unix(projectConfig.AccessGrantExpiration, 0)

		if time.Now().Before(expiration) {
			// Parse existing access grant
			access, err := uplink.ParseAccess(projectConfig.AccessGrant)
			if err == nil {
				return access, nil
			}

			// Existing access grant is invalid
			Log(LogOptions{
				Level: Warning,
				Str:   "Existing access grant is invalid. Regenerating...",
			})
		}
	} else {
		return nil, Log(LogOptions{
			Level:       Error,
			Str:         "Failed to get project config, please make sure this directory is a Quanta Control project",
			VerboseStr:  "Failed to get project config: %s",
			VerboseVars: []interface{}{err},
		})
	}

	projectId := projectConfig.ProjectID

	// Get new access grant from API
	res, err := http.Get(BuildURL(fmt.Sprintf("projects/%s/access_grant", projectId)))
	if err != nil {
		return nil, Log(LogOptions{
			Level:       Error,
			Str:         "Failed to authenticate with storage",
			VerboseStr:  "Failed to get access grant from API (request failed): %s",
			VerboseVars: []interface{}{err},
		})
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		// return nil, fmt.Errorf("failed to get access grant from Quanta Control servers (status: %s)", res.Status)
		return nil, Log(LogOptions{
			Level:       Error,
			Str:         "Failed to authenticate with storage",
			VerboseStr:  "Failed to get access grant from API (status: %s)",
			VerboseVars: []interface{}{res.Status},
		})
	}

	// Parse response
	var decodedRes models.AccessGrantResponse
	err = json.NewDecoder(res.Body).Decode(&decodedRes)
	if err != nil {
		return nil, Log(LogOptions{
			Level:       Error,
			Str:         "Failed to authenticate with storage",
			VerboseStr:  "Failed to parse API response: %s",
			VerboseVars: []interface{}{err},
		})
	}

	accessGrantStr := decodedRes.AccessGrant

	// Write access grant to project config
	projectConfig, err = WriteProjectConfig(".", models.ProjectFileData{
		AccessGrant: accessGrantStr,
		// TODO: Enforce this 24-hour expiration in Storj itself
		AccessGrantExpiration: time.Now().Add(time.Hour * 24).Unix(),
	})
	if err != nil {
		return nil, Log(LogOptions{
			Level: Error,
			Str:   "Failed to write project config: %s",
			Vars:  []interface{}{err},
		})
	}

	// Parse access grant string
	access, err := uplink.ParseAccess(accessGrantStr)
	if err != nil {
		return nil, Log(LogOptions{
			Level:       Error,
			Str:         "Failed to authenticate with storage",
			VerboseStr:  "Failed to parse Storj access: %s",
			VerboseVars: []interface{}{err},
		})
	}

	return access, nil
}

// Download objects in bulk from Storj.
//
// @param keys - List of object keys to download
//
// Returns an array of uplink download objects.
func DownloadBulk(keys []string) ([]*uplink.Download, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get access grant
	accessGrant, err := GetAccessGrant()
	if err != nil {
		return nil, err
	}

	// Open Storj project
	sp, err := uplink.OpenProject(ctx, accessGrant)
	if err != nil {
		return nil, err
	}

	// Download objects
	downloads := make([]*uplink.Download, len(keys))

	for _, key := range keys {
		d, err := sp.DownloadObject(ctx, config.I.Storage.Bucket, key, nil)
		if err != nil {
			return nil, err
		}
		defer d.Close()

		downloads = append(downloads, d)
	}

	return downloads, nil
}
