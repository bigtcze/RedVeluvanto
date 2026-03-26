package reddit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("test-agent/1.0")

	if c.userAgent != "test-agent/1.0" {
		t.Errorf("expected userAgent %q, got %q", "test-agent/1.0", c.userAgent)
	}
	if c.rateLimitRemaining != 60 {
		t.Errorf("expected rateLimitRemaining 60, got %d", c.rateLimitRemaining)
	}
	if c.httpClient == nil {
		t.Error("expected httpClient to be non-nil")
	}
}

func TestParseCommentChildren_Empty(t *testing.T) {
	result := parseCommentChildren(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 comments, got %d", len(result))
	}
}

func TestParseCommentChildren_SingleT1(t *testing.T) {
	commentData, _ := json.Marshal(rawComment{
		ID:     "abc123",
		Author: "testuser",
		Body:   "hello world",
		Score:  42,
		Depth:  0,
	})
	children := []redditThingChild{
		{Kind: "t1", Data: commentData},
	}

	result := parseCommentChildren(children)
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
	if result[0].ID != "abc123" {
		t.Errorf("expected ID %q, got %q", "abc123", result[0].ID)
	}
	if result[0].Author != "testuser" {
		t.Errorf("expected Author %q, got %q", "testuser", result[0].Author)
	}
	if result[0].Body != "hello world" {
		t.Errorf("expected Body %q, got %q", "hello world", result[0].Body)
	}
	if result[0].Score != 42 {
		t.Errorf("expected Score 42, got %d", result[0].Score)
	}
}

func TestParseCommentChildren_NonT1Filtered(t *testing.T) {
	commentData, _ := json.Marshal(rawComment{ID: "c1", Author: "user1", Body: "t1 comment"})
	postData, _ := json.Marshal(map[string]string{"id": "p1", "title": "a post"})
	moreData, _ := json.Marshal(map[string]interface{}{"id": "m1", "count": 5})

	children := []redditThingChild{
		{Kind: "t3", Data: postData},
		{Kind: "t1", Data: commentData},
		{Kind: "more", Data: moreData},
	}

	result := parseCommentChildren(children)
	if len(result) != 1 {
		t.Fatalf("expected 1 comment (t1 only), got %d", len(result))
	}
	if result[0].ID != "c1" {
		t.Errorf("expected ID %q, got %q", "c1", result[0].ID)
	}
}

func TestParseCommentChildren_NestedReplies(t *testing.T) {
	replyData, _ := json.Marshal(rawComment{
		ID:     "reply1",
		Author: "replier",
		Body:   "nested reply",
		Depth:  1,
	})
	repliesListing, _ := json.Marshal(redditListing{
		Kind: "Listing",
		Data: redditListingData{
			Children: []redditThingChild{
				{Kind: "t1", Data: replyData},
			},
		},
	})

	parentData, _ := json.Marshal(rawComment{
		ID:      "parent1",
		Author:  "op",
		Body:    "parent comment",
		Depth:   0,
		Replies: repliesListing,
	})

	children := []redditThingChild{
		{Kind: "t1", Data: parentData},
	}

	result := parseCommentChildren(children)
	if len(result) != 1 {
		t.Fatalf("expected 1 top-level comment, got %d", len(result))
	}
	if len(result[0].Replies) != 1 {
		t.Fatalf("expected 1 nested reply, got %d", len(result[0].Replies))
	}
	if result[0].Replies[0].ID != "reply1" {
		t.Errorf("expected reply ID %q, got %q", "reply1", result[0].Replies[0].ID)
	}
	if result[0].Replies[0].Author != "replier" {
		t.Errorf("expected reply author %q, got %q", "replier", result[0].Replies[0].Author)
	}
}

func TestParseCommentChildren_EmptyStringReplies(t *testing.T) {
	parentData, _ := json.Marshal(rawComment{
		ID:      "c1",
		Author:  "user",
		Body:    "no replies",
		Replies: json.RawMessage(`""`),
	})

	children := []redditThingChild{
		{Kind: "t1", Data: parentData},
	}

	result := parseCommentChildren(children)
	if len(result) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(result))
	}
	if len(result[0].Replies) != 0 {
		t.Errorf("expected 0 replies for empty string, got %d", len(result[0].Replies))
	}
}

func TestParseCommentChildren_MalformedData(t *testing.T) {
	children := []redditThingChild{
		{Kind: "t1", Data: json.RawMessage(`{invalid json}`)},
		{Kind: "t1", Data: json.RawMessage(`not even close`)},
	}

	result := parseCommentChildren(children)
	if len(result) != 0 {
		t.Errorf("expected 0 comments for malformed data, got %d", len(result))
	}
}

func TestDoRequest_SetsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "test-agent" {
			t.Errorf("expected User-Agent %q, got %q", "test-agent", got)
		}
		if got := r.Header.Get("Authorization"); got != "bearer test-token" {
			t.Errorf("expected Authorization %q, got %q", "bearer test-token", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-agent")
	resp, err := c.doRequest(context.Background(), http.MethodGet, server.URL+"/test", "test-token", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

func TestDoRequest_ParsesRateLimitHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Ratelimit-Remaining", "42.0")
		w.Header().Set("X-Ratelimit-Reset", "300")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-agent")
	resp, err := c.doRequest(context.Background(), http.MethodGet, server.URL+"/test", "", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if c.rateLimitRemaining != 42 {
		t.Errorf("expected rateLimitRemaining 42, got %d", c.rateLimitRemaining)
	}
	if c.rateLimitReset.IsZero() {
		t.Error("expected rateLimitReset to be set")
	}
}

func TestDoRequest_429Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("X-Ratelimit-Remaining", "0")
			w.Header().Set("X-Ratelimit-Reset", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("X-Ratelimit-Remaining", "50")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	c := NewClient("test-agent")
	resp, err := c.doRequest(context.Background(), http.MethodGet, server.URL+"/test", "", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if attempts < 2 {
		t.Errorf("expected at least 2 attempts (429 then retry), got %d", attempts)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 after retry, got %d", resp.StatusCode)
	}
}

func TestDoRequest_NoAuthHeaderWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("expected no Authorization header, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-agent")
	resp, err := c.doRequest(context.Background(), http.MethodGet, server.URL+"/test", "", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

func TestDoRequest_ContentTypeHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type %q, got %q", "application/x-www-form-urlencoded", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient("test-agent")
	resp, err := c.doRequest(context.Background(), http.MethodPost, server.URL+"/test", "tok", nil, "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

func TestDoRequest_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewClient("test-agent")
	_, err := c.doRequest(ctx, http.MethodGet, server.URL+"/test", "", nil, "")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
