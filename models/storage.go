package models

// NOTE: This file was copied from the server
// TODO: Use shared model files with the server

type PresignObjectData struct {
	// If true, the object will be uploaded as multiple parts.
	Multipart bool `json:"multipart"`
	// File size in bytes.
	// Used to determine amount of multipart upload presigned URLs to generate.
	Size int64 `json:"size"`
	// File MIME type
	ContentType string `json:"content_type"`
}

// Request body for `PresignMany` route.
type PresignManyRequest []struct {
	Method      string `json:"method"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Multipart   bool   `json:"multipart"`
	Size        int64  `json:"size"`
}

// Request body for `PresignOne` route.
type PresignOneRequest struct {
	Method      string `json:"method"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Multipart   bool   `json:"multipart"`
	Size        int64  `json:"size"`
}

// Response body for presign routes.
type PresignResponse struct {
	URLs     []string `json:"urls"`
	UploadID string   `json:"upload_id"`
}

type MultipartUploadPart struct {
	PartNumber int32  `json:"part_number"`
	ETag       string `json:"etag"`
}

// Request body for `CompleteMultipartUpload` route.
type CompleteMultipartUploadRequestBody struct {
	UploadId string                `json:"upload_id"`
	Key      string                `json:"key"`
	Parts    []MultipartUploadPart `json:"parts"`
}

// Request body for `AbortMultipartUpload` route.
type AbortMultipartUploadRequestBody struct {
	UploadId string `json:"upload_id"`
	Key      string `json:"key"`
}
