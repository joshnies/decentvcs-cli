package httpw

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/console"
)

// Send an HTTP request to the specified URL.
//
// @param method - HTTP method
//
// @param url - URL to send the request to
//
// @param body - Request body
//
// @param accessToken - Access token
//
// Returns the response object and any error that occurred.
func SendRequest(method string, url string, body *bytes.Buffer, accessToken string) (*http.Response, error) {
	// Check if POST request, and if so run custom logic
	if strings.ToUpper(method) == "POST" {
		return Post(url, body, accessToken)
	}

	// Build request
	httpClient := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Send request
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Validate response
	if err = validateResponse(res); err != nil {
		return nil, err
	}

	return res, nil
}

// Send a GET request to the specified URL.
//
// @param url - URL to send the request to
//
// @param body - Request body
//
// @param accessToken - Access token
//
// Returns the response object and any error that occurred.
func Get(url string, accessToken string) (*http.Response, error) {
	return SendRequest("GET", url, nil, accessToken)
}

// Send a DELETE request to the specified URL.
//
// @param url - URL to send the request to
//
// @param body - Request body
//
// @param accessToken - Access token
//
// Returns the response object and any error that occurred.
func Delete(url string, accessToken string) (*http.Response, error) {
	return SendRequest("DELETE", url, nil, accessToken)
}

// Send a POST request to the specified URL.
//
// @param method - HTTP method
//
// @param url - URL to send the request to
//
// @param body - Request body
//
// @param accessToken - Access token
//
// Returns the response object and any error that occurred.
func Post(url string, body *bytes.Buffer, accessToken string) (*http.Response, error) {
	// Build request
	httpClient := &http.Client{}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Send request
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Validate response
	if err = validateResponse(res); err != nil {
		return nil, err
	}

	return res, nil
}

// Validate HTTP response.
//
// @param res - HTTP response
func validateResponse(res *http.Response) error {
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
	}

	// Handle all other bad response status codes
	if res.StatusCode >= 300 {
		return console.Error(constants.ErrMsgInternal)
	}

	return nil
}
