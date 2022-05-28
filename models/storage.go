package models

// NOTE: This file was copied from qc-api
// TODO: Use shared model files with qc-api

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
type PresignManyRequestBody struct {
	Keys []string `json:"keys"`
}

// Request body for `PresignOne` route.
type PresignOneRequestBody struct {
	Key         string `json:"key"`
	Multipart   bool   `json:"multipart"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// Response body for `PresignOne` route.
type PresignOneResponse struct {
	// S3 multipart upload ID.
	// Only present if `multipart` is true and method is `PUT`.
	UploadID string `json:"upload_id"`
	// Presigned URLs for each part of the object
	URLs []string `json:"urls"`
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
