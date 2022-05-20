package httpw

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"

	"github.com/joshnies/qc/constants"
	"github.com/joshnies/qc/lib/console"
)

type RequestParams struct {
	URL           string
	Body          io.Reader
	AccessToken   string
	ContentType   string
	ContentLength int64
}

// Send an HTTP request to the specified URL.
//
// @param method - HTTP method
//
// @param params - Request params
//
// Returns the response object and any error that occurred.
//
func SendRequest(method string, params RequestParams) (*http.Response, error) {
	// Destructure input
	url := params.URL
	body := params.Body
	accessToken := params.AccessToken
	contentType := params.ContentType

	// Build request
	httpClient := &http.Client{}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Set headers
	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	} else if method == "POST" || method == "PUT" {
		// Add default JSON content type for POST & PUT methods
		req.Header.Add("Content-Type", "application/json")
	}

	if accessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}

	// Send request
	res, err := httpClient.Do(req)
	if err != nil {
		// Dump request
		dump, err := httputil.DumpRequestOut(req, false)
		if err != nil {
			console.ErrorPrintV("Failed to dump request: %s", err)
		}
		console.Verbose("Request:\n%s\n", string(dump))

		// Dump response
		if res != nil {
			dump, err = httputil.DumpResponse(res, true)
			if err != nil {
				console.ErrorPrintV("Failed to dump response: %s", err)
			}
			console.Verbose("Response:\n%s\n", string(dump))
		}

		return res, err
	}

	// Validate response
	if err = validateResponse(res); err != nil {
		// Dump request
		dump, err := httputil.DumpRequestOut(req, false)
		if err != nil {
			console.ErrorPrintV("Failed to dump request: %s", err)
		}
		console.Verbose("Request:\n%s\n", string(dump))

		// Dump response
		if res != nil {
			dump, err = httputil.DumpResponse(res, true)
			if err != nil {
				console.ErrorPrintV("Failed to dump response: %s", err)
			}
			console.Verbose("Response:\n%s\n", string(dump))
		}

		return res, err
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
	return SendRequest("GET", RequestParams{
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
	return SendRequest("DELETE", RequestParams{
		URL:         url,
		AccessToken: accessToken,
	})
}

// Send a POST request to the specified URL.
//
// @param params - Request params
//
// Returns the response object and any error that occurred.
//
func Post(params RequestParams) (*http.Response, error) {
	return SendRequest("POST", params)
}

// Send a PUT request to the specified URL.
//
// @param params - Request params
//
// Returns the response object and any error that occurred.
//
func Put(params RequestParams) (*http.Response, error) {
	return SendRequest("PUT", params)
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
	case http.StatusBadRequest:
		return console.Error("Bad request")
	}

	// Handle all other bad response status codes
	if res.StatusCode >= 300 {
		return console.Error(constants.ErrMsgInternal)
	}

	return nil
}
