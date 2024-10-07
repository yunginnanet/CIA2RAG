package anythingllm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	DefaultEndpoint = "http://localhost:3001/api/"
	auth            = "v1/auth"
)

type Config struct {
	Endpoint string
	APIKey   string
}

func NewConfig() *Config {
	return &Config{
		Endpoint: DefaultEndpoint,
	}
}

func (c *Config) WithEndpoint(endpoint string) *Config {
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	c.Endpoint = endpoint
	return c
}

func (c *Config) WithAPIKey(key string) *Config {
	if strings.TrimSpace(key) == "" {
		return c
	}
	c.APIKey = key
	return c
}

type AuthResponse struct {
	Authenticated bool   `json:"authenticated,omitempty"`
	Message       string `json:"message,omitempty"`
	err           error
	status        int
}

func (a *AuthResponse) processErr() error {
	if a.Authenticated && a.status == http.StatusOK {
		return nil
	}
	if a.status != http.StatusOK && a.Message == "" {
		a.Authenticated = false
		a.Message = http.StatusText(a.status)
	}
	if !a.Authenticated && a.Message != "" {
		return errors.New(a.Message)
	}
	return nil
}

func (a *AuthResponse) Err() error {
	return a.err
}

var ErrUnmarshal = errors.New("failed to unmarshal response")

func NewAuthResponse(res *http.Response) *AuthResponse {
	defer func() {
		_ = res.Body.Close()
	}()
	ar := &AuthResponse{}
	ar.status = res.StatusCode
	data, err := io.ReadAll(res.Body)
	if err != nil {
		ar.err = err
		return ar
	}
	if err = json.Unmarshal(data, ar); err != nil {
		ar.err = fmt.Errorf("%w: %v", ErrUnmarshal, err)
	}
	if ar.err == nil {
		ar.err = ar.processErr()
	}
	return ar
}

func (c *Config) get(endpoint string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.Endpoint+endpoint, nil)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.APIKey))
	}
	return http.DefaultClient.Do(req)
}

func (c *Config) post(endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, c.Endpoint+endpoint, body)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.APIKey))
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func (c *Config) Validate() error {
	res, err := c.get(auth)
	if err != nil {
		return err
	}
	ar := NewAuthResponse(res)
	return ar.Err()
}

type UploadLink struct {
	Link string `json:"link"`
}

type Document struct {
	ID                 string `json:"id"`
	Url                string `json:"url"`
	Title              string `json:"title"`
	DocAuthor          string `json:"docAuthor"`
	Description        string `json:"description"`
	DocSource          string `json:"docSource"`
	ChunkSource        string `json:"chunkSource"`
	Published          string `json:"published"`
	WordCount          int    `json:"wordCount"`
	PageContent        string `json:"pageContent"`
	TokenCountEstimate int    `json:"token_count_estimate"`
	Location           string `json:"location"`
}

type UploadLinkResponse struct {
	Success   bool        `json:"success"`
	Error     interface{} `json:"error"`
	Documents []Document  `json:"documents"`
}

func (c *Config) UploadLink(s string) (*Document, error) {
	l := &UploadLink{Link: s}
	dat, _ := json.Marshal(l)
	strings.NewReader(s)
	res, err := c.post("v1/document/upload-link", bytes.NewReader(dat))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		_ = res.Body.Close()
		return nil, fmt.Errorf("failed to upload link: %s", http.StatusText(res.StatusCode))
	}
	up := &UploadLinkResponse{}
	data, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if err = json.Unmarshal(data, up); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(up.Documents) == 0 {
		return nil, errors.New("no documents uploaded")
	}

	return &up.Documents[0], nil

}
