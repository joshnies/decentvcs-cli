package models

// Response to GET /projects/:id/access_grant
type AccessGrantResponse struct {
	AccessGrant string `json:"access_grant"`
}
