package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles interactions with Supabase Storage and Database.
type Client struct {
	baseURL    string
	anonKey    string
	serviceKey string
	httpClient *http.Client
}

// NewClient creates a new Supabase client.
func NewClient(baseURL, anonKey, serviceKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		anonKey:    anonKey,
		serviceKey: serviceKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateSignedUploadURL generates a signed URL for uploading a file to Supabase Storage.
func (c *Client) CreateSignedUploadURL(bucket, path string) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/upload/sign/%s/%s", c.baseURL, bucket, path)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create signed URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Return the full upload URL
	return fmt.Sprintf("%s/storage/v1%s", c.baseURL, result.URL), nil
}

// GetPublicURL returns the public URL for a stored file.
func (c *Client) GetPublicURL(bucket, path string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.baseURL, bucket, path)
}

// --- Database Operations (via Supabase REST API / PostgREST) ---

// InsertRecord inserts a record into a Supabase table.
func (c *Client) InsertRecord(table string, data interface{}) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/rest/v1/%s", c.baseURL, table)

	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("insert failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("supabase insert returned %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// QueryRecords queries records from a Supabase table with optional query parameters.
func (c *Client) QueryRecords(table, query string, authToken string) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/rest/v1/%s?%s", c.baseURL, table, query)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Use the user's token for RLS (Row Level Security)
	token := authToken
	if token == "" {
		token = c.serviceKey
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", c.anonKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("supabase query returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// UpdateRecord updates a record in a Supabase table.
func (c *Client) UpdateRecord(table, query string, data interface{}) error {
	url := fmt.Sprintf("%s/rest/v1/%s?%s", c.baseURL, table, query)

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase update returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
