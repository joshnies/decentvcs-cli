package models

// Structure for data written to the global config file for user.
type GlobalConfig struct {
	Auth GlobalConfigAuth `json:"auth"`
}

type GlobalConfigAuth struct {
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	IDToken         string `json:"id_token"`
	ExpiresIn       int64  `json:"expires_in"`
	AuthenticatedAt int64  `json:"authenticated_at"`
}
