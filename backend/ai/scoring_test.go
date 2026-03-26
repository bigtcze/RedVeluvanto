package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func liteLLMResponse(content string) []byte {
	resp := ChatResponse{
		Choices: []struct {
			Message Message `json:"message"`
		}{
			{Message: Message{Role: "assistant", Content: content}},
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

func newTestClient(serverURL string) *Client {
	return NewClient(serverURL, "test-key", "test-model")
}

func TestScoreThread_ValidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(liteLLMResponse(`{"score": 85, "reason": "good match"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.ScoreThread(context.Background(), "title", "body", "golang", "Go programming", "concurrency", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score != 85 {
		t.Errorf("expected score 85, got %d", result.Score)
	}
	if result.Reason != "good match" {
		t.Errorf("expected reason %q, got %q", "good match", result.Reason)
	}
}

func TestScoreThread_MarkdownCodeBlock(t *testing.T) {
	content := "```json\n{\"score\": 72, \"reason\": \"wrapped in markdown\"}\n```"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(liteLLMResponse(content))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.ScoreThread(context.Background(), "t", "b", "sub", "desc", "kw", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score != 72 {
		t.Errorf("expected score 72, got %d", result.Score)
	}
	if result.Reason != "wrapped in markdown" {
		t.Errorf("expected reason %q, got %q", "wrapped in markdown", result.Reason)
	}
}

func TestScoreThread_ScoreZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(liteLLMResponse(`{"score": 0, "reason": "irrelevant"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.ScoreThread(context.Background(), "t", "b", "sub", "desc", "kw", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score != 0 {
		t.Errorf("expected score 0, got %d", result.Score)
	}
}

func TestScoreThread_Score100(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(liteLLMResponse(`{"score": 100, "reason": "perfect"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.ScoreThread(context.Background(), "t", "b", "sub", "desc", "kw", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score != 100 {
		t.Errorf("expected score 100, got %d", result.Score)
	}
}

func TestScoreThread_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(liteLLMResponse("this is not json at all"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ScoreThread(context.Background(), "t", "b", "sub", "desc", "kw", nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parsing scoring result") {
		t.Errorf("expected parsing error, got: %v", err)
	}
}

func TestScoreThread_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ChatResponse{Choices: nil}
		b, _ := json.Marshal(resp)
		w.Write(b)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ScoreThread(context.Background(), "t", "b", "sub", "desc", "kw", nil)
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Errorf("expected 'no choices' error, got: %v", err)
	}
}

func TestScoreThread_HTTP500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ScoreThread(context.Background(), "t", "b", "sub", "desc", "kw", nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status 500 in error, got: %v", err)
	}
}

func TestScoreThread_WithProductContext(t *testing.T) {
	var capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		capturedBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.Write(liteLLMResponse(`{"score": 55, "reason": "product relevant"}`))
	}))
	defer server.Close()

	product := &ProductContext{
		Name:            "TestProduct",
		Description:     "A cool product",
		TargetAudience:  "developers",
		Differentiators: "fast and free",
	}
	client := newTestClient(server.URL)
	result, err := client.ScoreThread(context.Background(), "title", "body", "sub", "desc", "kw", product)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score != 55 {
		t.Errorf("expected score 55, got %d", result.Score)
	}

	if !strings.Contains(capturedBody, "TestProduct") {
		t.Error("request body should contain product name")
	}
	if !strings.Contains(capturedBody, "developers") {
		t.Error("request body should contain target audience")
	}
	if !strings.Contains(capturedBody, "fast and free") {
		t.Error("request body should contain differentiators")
	}
}

func TestScoreThread_WithoutProductContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(liteLLMResponse(`{"score": 40, "reason": "generic match"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.ScoreThread(context.Background(), "t", "b", "sub", "desc", "kw", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score != 40 {
		t.Errorf("expected score 40, got %d", result.Score)
	}
}
