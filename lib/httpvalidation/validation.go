package httpvalidation

import (
	"net/http"

	"github.com/joshnies/quanta/constants"
	"github.com/joshnies/quanta/lib/console"
)

// Validate HTTP response.
//
// @param res - HTTP response
//
// Returns any error that occurred.
//
func ValidateResponse(res *http.Response) error {
	// Check response status
	switch res.StatusCode {
	case http.StatusUnauthorized:
		return console.Error("Unauthorized")
	case http.StatusNotFound:
		return console.Error("Resource not found")
	case http.StatusRequestTimeout:
		return console.Error("HTTP request timed out")
	case http.StatusConflict:
		return console.Error("Resource already exists")
	case http.StatusBadRequest:
		return console.Error("Bad request")
	}

	// Handle all other bad response status codes
	if res.StatusCode >= 300 {
		return console.Error(constants.ErrMsgInternal)
	}

	return nil
}
