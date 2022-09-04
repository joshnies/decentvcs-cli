package httpvalidation

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/decentvcs/cli/lib/console"
)

// Validate HTTP response.
//
// @param res - HTTP response
//
// Returns any error that occurred.
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
		if errMsg, ok := resBody["error"]; ok {
			msg = errMsg.(string)
		} else {
			resBodyBytes, _ := json.MarshalIndent(resBody, "", "  ")
			console.ErrorPrintV("Server response:\n%v", string(resBodyBytes))
		}

		return errors.New(msg)
	}

	return nil
}
