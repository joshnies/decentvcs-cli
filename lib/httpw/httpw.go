package httpw

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/console"
)

// Send a GET request to the specified URL.
func Get(url string, accessToken string) (*http.Response, error) {
	// Build request
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	// Send request
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Check response status
	switch res.StatusCode {
	case http.StatusUnauthorized:
		return nil, console.Error("Unauthorized")
	case http.StatusNotFound:
		return nil, console.Error("Resource not found")
	case http.StatusRequestTimeout:
		return nil, console.Error("HTTP request timed out")
	case http.StatusBadRequest:
		return nil, console.Error(constants.ErrMsgInternal)
	case http.StatusInternalServerError:
		return nil, console.Error(constants.ErrMsgInternal)
	case http.StatusServiceUnavailable:
		return nil, console.Error(constants.ErrMsgInternal)
	}

	return res, nil
}

// Send a POST request to the specified URL.
func Post(url string, data []byte, accessToken string) (*http.Response, error) {
	// Build request
	httpClient := &http.Client{}
	body := bytes.NewBuffer(data)
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

	// Check response status
	switch res.StatusCode {
	case http.StatusUnauthorized:
		return nil, console.Error("Unauthorized")
	case http.StatusNotFound:
		return nil, console.Error("Resource not found")
	case http.StatusRequestTimeout:
		return nil, console.Error("HTTP request timed out")
	case http.StatusBadRequest:
		return nil, console.Error(constants.ErrMsgInternal)
	case http.StatusInternalServerError:
		return nil, console.Error(constants.ErrMsgInternal)
	case http.StatusServiceUnavailable:
		return nil, console.Error(constants.ErrMsgInternal)
	}

	return res, nil
}
