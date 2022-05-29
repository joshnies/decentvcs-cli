package httpvalidation

import (
	"fmt"
	"net/http"
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
		return fmt.Errorf("unauthorized")
	case http.StatusNotFound:
		return fmt.Errorf("resource not found")
	case http.StatusRequestTimeout:
		return fmt.Errorf("request timed out")
	case http.StatusConflict:
		return fmt.Errorf("resource already exists")
	case http.StatusBadRequest:
		return fmt.Errorf("bad request")
	}

	// Handle all other bad response status codes
	if res.StatusCode >= 300 {
		return fmt.Errorf("received http status: %s", res.Status)
	}

	return nil
}
