package models

// Structure for data written to QC auth file for user.
type GlobalConfig struct {
	Auth GlobalConfigAuth `json:"auth"`
}

type GlobalConfigAuth struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
}
