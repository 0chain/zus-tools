package transport

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type HTTPTransport struct {
	Name    string
	BaseURL string
	Timeout int
}

func NewHTTPTransport(name, baseURL string, timeout int) *HTTPTransport {
	return &HTTPTransport{
		Name:    name,
		BaseURL: baseURL,
		Timeout: timeout,
	}
}

func (t *HTTPTransport) GetInfo() string {
	return "HTTP Transport: " + t.Name + ", BaseURL: " + t.BaseURL
}

// POSTRequest sends a HTTP POST request to the specified endpoint with the given data.
func (t *HTTPTransport) POSTRequest(endpoint string, data []byte) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Duration(t.Timeout) * time.Second,
	}

	url, err := url.JoinPath(t.BaseURL, endpoint)
	if err != nil {
		return nil, fmt.Errorf("error forming URL: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set required headers (can be extended)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	log.Printf("Status: %s\nResponse: %s", resp.Status, string(bodyBytes))
	return bodyBytes, nil
}

// GETRequest sends a HTTP GET request to the specified endpoint.
func (t *HTTPTransport) GETRequest(endpoint string) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Duration(t.Timeout) * time.Second,
	}

	url, err := url.JoinPath(t.BaseURL, endpoint)
	if err != nil {
		return nil, fmt.Errorf("error forming URL: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating GET request: %v", err)
	}

	// Optional headers (you can extend this)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET request to %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading GET response: %v", err)
	}

	log.Printf("Status: %s\nResponse: %s", resp.Status, string(bodyBytes))
	return bodyBytes, nil
}
