package constants

// Config
const VerboseEnvVar = "V"

// File system
const GlobalConfigFileName = "~/.decent/config.json"

// Error messages
const ErrMsgInternal = "An internal error occurred. If the issue persists, please contact us."
const ErrMsgAuthFailed = "Authentication failed"
const ErrMsgNotAuthenticated = "Not logged in. You can use `decent login` to authenticate."

// Formatting
const TimeFormat = "2006-01-02 @ 03:04:05pm"

// Auth0
const Auth0DomainDev = "https://dev--4bueuyj.us.auth0.com"
const Auth0ClientIDDev = "4pBLr8bNSlrYHHb4fIgO8I11KMnj3f5X"

// Filebase
const SlowDownFileContents = "<?xml version=\"1.0\" encoding=\"UTF-8\"?><Error><Code>SlowDown</Code><Message>Please reduce your request rate.</Message><RequestId/><HostId/></Error>"

// zstd
const ZstdHeader = "28b52ffd"
