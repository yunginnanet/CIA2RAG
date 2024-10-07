package anythingllm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

const (
	DefaultEndpoint = "http://localhost:3001/api/"
	auth            = "v1/auth"
)

type Config struct {
	Endpoint   string
	APIKey     string
	Workspace  string
	seen       Seen
	forceEmbed bool
	mu         sync.RWMutex
}

func NewConfig() *Config {
	c := &Config{
		Endpoint: DefaultEndpoint,
		seen:     make(Seen),
	}
	return c
}

func (c *Config) WithWorkspace(workspace string) *Config {
	c.Workspace = workspace
	return c
}

func (c *Config) WithForceEmbed(force bool) *Config {
	c.forceEmbed = force
	return c
}

func (c *Config) hasSeenURL(s string) bool {
	if strings.HasPrefix(s, "link://") {
		s = s[7:]
	}
	c.mu.RLock()
	_, ok := c.seen["link://"+s]
	if !ok {
		_, ok = c.seen[s]
	}
	c.mu.RUnlock()
	return ok
}

func (c *Config) markSeenURL(s string) {
	if strings.HasPrefix(s, "link://") {
		s = s[7:]
	}
	c.mu.Lock()
	c.seen["link://"+s] = true
	c.mu.Unlock()
}

func (c *Config) updateSeen() error {
	docsFolder, err := c.GetDocuments()
	if err != nil {
		return err
	}
	for _, items := range docsFolder {
		for _, item := range items {
			if item.Type == "file" {
				if !strings.Contains(item.ChunkSource, "cia.gov") {
					continue
				}
				c.markSeenURL(item.ChunkSource)
				if c.forceEmbed {
					log.Printf("[info] force re-embedding document: %s", item.ChunkSource)
					if err := c.AddDocumentItem(&item); err != nil {
						log.Printf("[err] failed to add document: %v", err)
					}
				}
			}
		}
	}
	return nil
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

func (c *Config) delete(endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, c.Endpoint+endpoint, body)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.APIKey))
	}
	req.Header.Set("Content-Type", "application/json")
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
	req.Header.Set("Accept", "application/json")
	return http.DefaultClient.Do(req)
}

func (c *Config) Validate() error {
	res, err := c.get(auth)
	if err != nil {
		return err
	}
	ar := NewAuthResponse(res)
	if err := ar.Err(); err != nil {
		return err
	}
	return c.updateSeen()
}
