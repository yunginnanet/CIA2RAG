package anythingllm

import (
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
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.APIKey))
	return http.DefaultClient.Do(req)
}

func (c *Config) post(endpoint string, body io.Reader) (*http.Response, error) {
	req, _ := http.NewRequest(http.MethodPost, c.Endpoint+endpoint, body)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.APIKey))
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
