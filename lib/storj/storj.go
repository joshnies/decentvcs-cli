package storj

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"time"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/lib/projects"
	"github.com/joshnies/qc-cli/models"
	"storj.io/uplink"
)

// Get Storj access grant for project.
// Gives read-write access to project folder in Storj bucket.
//
// Returns Storj access grant (unserialized).
func GetAccessGrant() (*uplink.Access, error) {
	gc := auth.Validate()

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
			console.Warning("Existing access grant is invalid. Regenerating...")
		}
	} else {
		return nil, console.Error("Failed to get project config, please make sure this directory is a Quanta Control project. Error: %v", err)
	}

	projectId := projectConfig.ProjectID

	// Get new access grant from API
	apiUrl := api.BuildURLf("projects/%s/access_grant?perm=w", projectId)
	res, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return nil, console.Error("Failed to authenticate with storage: %v", err)
	}
	defer res.Body.Close()

	// Parse response
	var decodedRes models.AccessGrantResponse
	err = json.NewDecoder(res.Body).Decode(&decodedRes)
	if err != nil {
		return nil, console.Error("Failed to authenticate with storage: %v", err)
	}

	accessGrantStr := decodedRes.AccessGrant

	// Write access grant to project config
	projectConfig, err = projects.WriteProjectConfig(".", models.ProjectConfig{
		AccessGrant: accessGrantStr,
		// TODO: Enforce this 24-hour expiration in Storj itself
		AccessGrantExpiration: time.Now().Add(time.Hour * 24).Unix(),
	})
	if err != nil {
		return nil, console.Error("Failed to write project config: %v", err)
	}

	// Parse access grant string
	access, err := uplink.ParseAccess(accessGrantStr)
	if err != nil {
		return nil, console.Error("Failed to authenticate with storage: %v", err)
	}

	return access, nil
}

// Download objects in bulk from Storj.
//
// @param keys - List of object keys to download
//
// Returns an array of uplink download objects.
func DownloadBulk(projectConfig models.ProjectConfig, keys []string) ([]*uplink.Download, error) {
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
	defer sp.Close()

	// Download objects
	// TODO: Download objects in parallel
	var downloads []*uplink.Download

	for _, key := range keys {
		d, err := sp.DownloadObject(ctx, config.I.Storage.Bucket, projectConfig.ProjectID+"/"+key, nil)
		if err != nil && !errors.Is(err, uplink.ErrObjectNotFound) {
			return nil, err
		}

		if d != nil {
			defer d.Close()
			downloads = append(downloads, d)
		}
	}

	return downloads, nil
}

// Upload objects in bulk to Storj.
//
// @param paths - Paths to files that will be uploaded
//
func UploadBulk(prefix string, paths []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get access grant
	accessGrant, err := GetAccessGrant()
	if err != nil {
		return err
	}

	// Open Storj project
	sp, err := uplink.OpenProject(ctx, accessGrant)
	if err != nil {
		return err
	}
	defer sp.Close()

	for _, path := range paths {
		// Read file
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// Start upload
		upload, err := sp.UploadObject(ctx, config.I.Storage.Bucket, prefix+"/"+path, nil)
		if err != nil {
			return err
		}

		// Copy file data to upload buffer
		buf := bytes.NewBuffer(data)
		_, err = io.Copy(upload, buf)
		if err != nil {
			_ = upload.Abort()
			return console.Error("Failed to upload file: %v", err)
		}

		// Commit upload
		err = upload.Commit()
		if err != nil {
			return err
		}
	}

	return nil
}

// Upload JSON data as new object to Storj.
//
// @param key - Key of object to upload
// @param data - JSON data as bytes
//
func UploadJSON(key string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get access grant
	accessGrant, err := GetAccessGrant()
	if err != nil {
		return err
	}

	// Open Storj project
	sp, err := uplink.OpenProject(ctx, accessGrant)
	if err != nil {
		return err
	}
	defer sp.Close()

	// Start upload
	upload, err := sp.UploadObject(ctx, config.I.Storage.Bucket, key, nil)
	if err != nil {
		return err
	}

	// Copy file data to upload buffer
	buf := bytes.NewBuffer(data)
	_, err = io.Copy(upload, buf)
	if err != nil {
		_ = upload.Abort()
		return console.Error("Failed to upload file: %v", err)
	}

	// Commit upload
	return upload.Commit()
}
