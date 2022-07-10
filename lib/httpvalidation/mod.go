package httpvalidation

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joshnies/decent/lib/console"
)

// Validate HTTP response.
//
// @param res - HTTP response
//
// Returns any error that occurred.
//
func ValidateResponse(res *http.Response) error {
	var msg string

	// Check response status
	switch res.StatusCode {
	case http.StatusUnauthorized:
		msg = "unauthorized"
	case http.StatusNotFound:
		msg = "resource not found"
	case http.StatusRequestTimeout:
		msg = "request timed out"
	case http.StatusConflict:
		msg = "resource already exists"
	case http.StatusBadRequest:
		msg = "bad request"
	}

	// Handle all other bad response status codes
	if res.StatusCode >= 300 {
		msg = fmt.Sprintf("received http status %d", res.StatusCode)
	}

	if msg != "" {
		// Parse response body
		var resBody map[string]interface{}
		json.NewDecoder(res.Body).Decode(&resBody)

		// Print error message
		if resMsg, ok := resBody["message"]; ok {
			msg = resMsg.(string)
		} else {
			resBodyBytes, _ := json.MarshalIndent(resBody, "", "  ")
			console.ErrorPrintV("Server response:\n%v", string(resBodyBytes))
		}

		return fmt.Errorf(msg)
	}

	return nil
}
