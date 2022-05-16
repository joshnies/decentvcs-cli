package storj

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/lib/util"
	"github.com/joshnies/qc/models"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
	"golang.org/x/exp/maps"
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
	if err != nil {
		return nil, console.Error("Failed to get project config, please make sure this directory is a Quanta Control project. Error: %v", err)
	}

	projectId := projectConfig.ProjectID

	// Get project from database
	apiUrl := api.BuildURLf("projects/%s", projectId)
	projectRes, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return nil, err
	}
	defer projectRes.Body.Close()

	// Parse response
	var project models.Project
	err = json.NewDecoder(projectRes.Body).Decode(&project)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Unix(project.StorjAccessGrantExpiresAt, 0)

	if time.Now().Before(expiresAt) {
		// Parse existing access grant
		access, err := uplink.ParseAccess(project.StorjAccessGrant)
		if err == nil {
			return access, nil
		}

		// Existing access grant is invalid
		console.Warning("Existing access grant is invalid. Regenerating...")
	}

	// Get new access grant from API
	apiUrl = api.BuildURLf("projects/%s/access_grant?perm=w", projectId)
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

	// Parse access grant string
	accessGrantStr := decodedRes.AccessGrant
	access, err := uplink.ParseAccess(accessGrantStr)
	if err != nil {
		return nil, console.Error("Failed to authenticate with storage: %v", err)
	}

	// Update project with new access grant and expiration
	apiUrl = api.BuildURLf("projects/%s", projectId)
	projectUpdateData := models.Project{
		StorjAccessGrant: accessGrantStr,
		// TODO: Enforce this 24-hour expiration in Storj itself
		StorjAccessGrantExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
	}
	body := bytes.NewBuffer([]byte{})
	err = json.NewEncoder(body).Encode(projectUpdateData)
	if err != nil {
		return nil, err
	}
	_, err = httpw.Post(apiUrl, body, gc.Auth.AccessToken)
	if err != nil {
		return nil, err
	}

	return access, nil
}

// Download a single object from Storj.
//
// @param key - Key of object to download
//
// Returns object data.
func Download(projectId string, key string) ([]byte, error) {
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

	// Download object
	key = fmt.Sprintf("%s/%s", projectId, key)
	d, err := sp.DownloadObject(ctx, config.I.Storage.Bucket, key, nil)
	if err != nil && !errors.Is(err, uplink.ErrObjectNotFound) {
		return nil, err
	}
	if d == nil {
		return nil, console.Error("Failed to download object %s", key)
	}
	defer d.Close()

	// Read object
	buf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buf, d)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Download objects in bulk from Storj.
//
// @param projectId - Project ID
// @param keys - List of object keys to download
//
// Returns:
//
// - map of object key to object data.
//
// - any error that occurred.
func DownloadBulk(projectId string, keys []string) (map[string][]byte, error) {
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
	dataMap := map[string][]byte{}

	bar := util.NewProgressBar(int64(len(keys)), "Downloading objects")
	for _, key := range keys {
		// Download object
		d, err := sp.DownloadObject(ctx, config.I.Storage.Bucket, projectId+"/"+key, nil)
		if err != nil && !errors.Is(err, uplink.ErrObjectNotFound) {
			return nil, err
		}
		if d == nil {
			return nil, console.Error("Failed to download object %s", key)
		}
		defer d.Close()

		// Read object
		buf := bytes.NewBuffer([]byte{})
		_, err = io.Copy(buf, d)
		if err != nil {
			return nil, err
		}

		// Store object
		dataMap[key] = buf.Bytes()
		bar.Increment()
	}
	bar.Wait()

	return dataMap, nil
}

// Upload objects in bulk to Storj.
//
// @param prefix - Prefix to use for uploaded objects
// @param hashMap - Map of file path to hash
//
// Returns any error that occurred.
func UploadBulk(prefix string, hashMap map[string]string) error {
	ctx, cancel := context.WithCancel(context.Background())
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

	filePaths := maps.Keys(hashMap)

	// Upload objects in parallel, but in chunks
	chunks := util.ChunkMap(hashMap, 32)
	for _, chunk := range chunks {
		var wg sync.WaitGroup
		wg.Add(len(filePaths))

		p := mpb.New(mpb.WithWidth(60))

		for path, hash := range chunk {
			// TODO: Keep list of upload objects, commit all at once
			go uploadRoutine(ctx, sp, path, prefix+"/"+hash, &wg, p)
		}

		wg.Wait()
		p.Wait()
	}

	return nil
}

func uploadRoutine(ctx context.Context, sp *uplink.Project, path string, key string, wg *sync.WaitGroup, p *mpb.Progress) {
	defer wg.Done()

	// Read file
	file, err := os.Open(path)
	if err != nil {
		console.ErrorPrint("Failed to open file %s: %v", path, err)
		panic(err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		console.ErrorPrint("Failed to stat file %s: %v", path, err)
		panic(err)
	}

	total := fileInfo.Size()

	bar := p.New(total,
		mpb.BarStyle(),
		mpb.PrependDecorators(
			decor.CountersKibiByte("% .2f / % .2f "),
		),
		mpb.AppendDecorators(
			decor.Name(filepath.Base(path), decor.WC{W: 20, C: decor.DidentRight}),
			decor.Name(" | "),
			decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
		),
	)
	proxyReader := bar.ProxyReader(file)
	defer proxyReader.Close()

	// Start upload
	upload, err := sp.UploadObject(ctx, config.I.Storage.Bucket, key, nil)
	if err != nil {
		console.ErrorPrint("Failed to start Storj upload for key \"%s\"", key)
		panic(err)
	}

	// Copy file data to upload buffer
	_, err = io.Copy(upload, proxyReader)
	if err != nil {
		_ = upload.Abort()
		console.ErrorPrint("Failed to copy object data to Storj upload buffer; object key: \"%s\"", key)
		panic(err)
	}

	// Add path to object metadata
	// TODO: Is this needed?
	upload.SetCustomMetadata(ctx, uplink.CustomMetadata{
		"path": path,
	})

	// Commit upload
	fmt.Println("Committing upload...")
	err = upload.Commit()
	if err != nil {
		console.ErrorPrint("Failed to commit Storj upload for file \"%s\", key \"%s\"", path, key)
		panic(err)
	}
	fmt.Println("Done")
}

// Upload a single object as bytes to Storj.
//
// @param key - Key of object to upload
// @param data - Object data as bytes
//
func UploadBytes(key string, data []byte) error {
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

// Upload objects as bytes to Storj in bulk.
//
// @param dataMap - Map of object keys to object data
//
func UploadBytesBulk(prefix string, dataMap map[string][]byte) error {
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

	for _, path := range maps.Keys(dataMap) {
		// Start upload
		upload, err := sp.UploadObject(ctx, config.I.Storage.Bucket, prefix+"/"+path, nil)
		if err != nil {
			return err
		}

		// Copy file data to upload buffer
		buf := bytes.NewBuffer(dataMap[path])
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
