package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ciascrape/pkg/anythingllm"
)

func TestNewConfig_SetsDefaultValues(t *testing.T) {
	config := NewConfig("testCollection")
	if config.Collection != "testCollection" {
		t.Errorf("expected collection to be 'testCollection', got %s", config.Collection)
	}
	if config.MaxPages != defaultMaxPages {
		t.Errorf("expected max pages to be %d, got %d", defaultMaxPages, config.MaxPages)
	}
	if config.AnythingLLM == nil {
		t.Errorf("expected AnythingLLM to be initialized, got nil")
	}
}

func TestWithMaxPages_SetsMaxPages(t *testing.T) {
	config := NewConfig("testCollection").WithMaxPages(100)
	if config.MaxPages != 100 {
		t.Errorf("expected max pages to be 100, got %d", config.MaxPages)
	}
}

func TestWithAnythingLLM_SetsAnythingLLMConfig(t *testing.T) {
	anythingLLMConfig := anythingllm.NewConfig().WithEndpoint("testEndpoint").WithAPIKey("testKey")
	config := NewConfig("testCollection").WithAnythingLLM(anythingLLMConfig)
	if config.AnythingLLM.Endpoint != "testEndpoint/" {
		t.Errorf("expected endpoint to be 'testEndpoint/', got %s", config.AnythingLLM.Endpoint)
	}
	if config.AnythingLLM.APIKey != "testKey" {
		t.Errorf("expected API key to be 'testKey', got %s", config.AnythingLLM.APIKey)
	}
}

func TestValidate_ReturnsErrorForMissingCollection(t *testing.T) {
	config := NewConfig("").WithMaxPages(10)
	err := config.Validate()
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected error %v, got %v", ErrInvalidConfig, err)
	}
}

func TestValidate_ReturnsErrorForInvalidMaxPages(t *testing.T) {
	config := NewConfig("testCollection").WithMaxPages(0)
	err := config.Validate()
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected error %v, got %v", ErrInvalidConfig, err)
	}
}

func TestValidate_ReturnsErrorForInvalidAnythingLLMConfig(t *testing.T) {
	anythingLLMConfig := anythingllm.NewConfig().WithEndpoint("").WithAPIKey("")
	config := NewConfig("testCollection").WithAnythingLLM(anythingLLMConfig)
	err := config.Validate()
	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected error %v, got %v", ErrInvalidConfig, err)
	}
}

func TestValidate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer testKey" {
			_, _ = w.Write([]byte(`{"authenticated": true}`))
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(`{"message": "Invalid API Key"}`))
		w.WriteHeader(http.StatusUnauthorized)
	}))
	anythingLLMConfig := anythingllm.NewConfig().WithEndpoint(server.URL).WithAPIKey("testKey")
	config := NewConfig("stargate").WithAnythingLLM(anythingLLMConfig).WithMaxPages(10)
	err := config.Validate()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	anythingLLMConfig = anythingllm.NewConfig().WithEndpoint(server.URL).WithAPIKey("yeet")
	config = NewConfig("stargate").WithAnythingLLM(anythingLLMConfig).WithMaxPages(10)
	err = config.Validate()
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Invalid API Key") {
		t.Errorf("expected error to contain 'Invalid API Key', got %v", err)
	}
}
