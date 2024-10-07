package anythingllm

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestNewAuthResponse_Success(t *testing.T) {
	body := bytes.NewBufferString(`{"authenticated": true, "message": ""}`)
	res := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(body),
	}
	authResponse := NewAuthResponse(res)
	if authResponse.Err() != nil {
		t.Errorf("expected no error, got %v", authResponse.Err())
	}
	if !authResponse.Authenticated {
		t.Errorf("expected authenticated to be true, got false")
	}
}

func TestNewAuthResponse_UnmarshalError(t *testing.T) {
	body := bytes.NewBufferString(`invalid json`)
	res := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(body),
	}
	authResponse := NewAuthResponse(res)
	if authResponse.Err() == nil {
		t.Errorf("expected error, got nil")
	}
	if !errors.Is(authResponse.Err(), ErrUnmarshal) {
		t.Errorf("expected unmarshal error, got %v", authResponse.Err())
	}
}

func TestNewAuthResponse_AuthenticationFailed(t *testing.T) {
	body := bytes.NewBufferString(`{"authenticated": false, "message": "authentication failed"}`)
	res := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(body),
	}
	authResponse := NewAuthResponse(res)
	if authResponse.Err() == nil {
		t.Fatal("expected error, got nil")
	}
	if authResponse.Err().Error() != "authentication failed" {
		t.Errorf("expected 'authentication failed', got %v", authResponse.Err().Error())
	}
}

func TestProcessErr_ReturnsNilForAuthenticated(t *testing.T) {
	authResponse := &AuthResponse{
		Authenticated: true,
		status:        http.StatusOK,
	}
	err := authResponse.processErr()
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestProcessErr_ReturnsErrorForUnauthenticated(t *testing.T) {
	authResponse := &AuthResponse{
		Authenticated: false,
		status:        http.StatusUnauthorized,
		Message:       "unauthorized",
	}
	err := authResponse.processErr()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "unauthorized" {
		t.Errorf("expected 'unauthorized', got %v", err.Error())
	}
}
