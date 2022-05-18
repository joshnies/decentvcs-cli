package httpw

import (
	"fmt"
	"io"
	"net/http"

	"github.com/joshnies/qc/constants"
	"github.com/joshnies/qc/lib/console"
)

type RequestInput struct {
	URL         string
	Body        io.Reader
	AccessToken string
	ContentType string
}

// Send an HTTP request to the specified URL.
//
// @param method - HTTP method
//
// @param input - Request input
//
// Returns the response object and any error that occurred.
//
func SendRequest(method string, input RequestInput) (*http.Response, error) {
	// Destructure input
	url := input.URL
	body := input.Body
	accessToken := input.AccessToken
	contentType := input.ContentType

	// Build request
	httpClient := &http.Client{}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Set headers
	if contentType != "" {
		req.Header.Add("Content-Type", "application/json")
	}

	if accessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}

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
// @param accessToken - Access token
//
// Returns the response object and any error that occurred.
//
func Get(url string, accessToken string) (*http.Response, error) {
	return SendRequest("GET", RequestInput{
		URL:         url,
		AccessToken: accessToken,
	})
}

// Send a DELETE request to the specified URL.
//
// @param url - URL to send the request to
//
// @param accessToken - Access token
//
// Returns the response object and any error that occurred.
//
func Delete(url string, accessToken string) (*http.Response, error) {
	return SendRequest("DELETE", RequestInput{
		URL:         url,
		AccessToken: accessToken,
	})
}

// Send a POST request to the specified URL.
//
// @param input - Request input
//
// Returns the response object and any error that occurred.
//
func Post(input RequestInput) (*http.Response, error) {
	return SendRequest("POST", input)
}

// Send a PUT request to the specified URL.
//
// @param input - Request input
//
// Returns the response object and any error that occurred.
//
func Put(input RequestInput) (*http.Response, error) {
	return SendRequest("PUT", input)
}

// Validate HTTP response.
//
// @param res - HTTP response
//
// Returns any error that occurred.
//
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
