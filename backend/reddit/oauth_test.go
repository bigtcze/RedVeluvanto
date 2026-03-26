package reddit

import (
	"net/url"
	"strings"
	"testing"
)

func newTestConfig() *OAuthConfig {
	return &OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURI:  "http://localhost:8090/api/reddit/callback",
		UserAgent:    "TestAgent/1.0",
	}
}

func TestAuthorizeURL_BaseURL(t *testing.T) {
	cfg := newTestConfig()
	result := cfg.AuthorizeURL("test-state")

	if !strings.HasPrefix(result, "https://www.reddit.com/api/v1/authorize?") {
		t.Errorf("URL should start with Reddit authorize endpoint, got: %s", result)
	}
}

func TestAuthorizeURL_RequiredParams(t *testing.T) {
	cfg := newTestConfig()
	rawURL := cfg.AuthorizeURL("my-state")

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("AuthorizeURL returned invalid URL: %v", err)
	}

	params := parsed.Query()

	tests := []struct {
		param string
		value string
	}{
		{"client_id", "test-client-id"},
		{"response_type", "code"},
		{"state", "my-state"},
		{"redirect_uri", "http://localhost:8090/api/reddit/callback"},
		{"duration", "permanent"},
		{"scope", "identity read submit"},
	}

	for _, tt := range tests {
		t.Run(tt.param, func(t *testing.T) {
			got := params.Get(tt.param)
			if got != tt.value {
				t.Errorf("param %q = %q, want %q", tt.param, got, tt.value)
			}
		})
	}
}

func TestAuthorizeURL_DifferentStates(t *testing.T) {
	cfg := newTestConfig()
	url1 := cfg.AuthorizeURL("state-aaa")
	url2 := cfg.AuthorizeURL("state-bbb")

	if url1 == url2 {
		t.Error("different state values should produce different URLs")
	}
	if !strings.Contains(url1, "state-aaa") {
		t.Error("URL should contain state-aaa")
	}
	if !strings.Contains(url2, "state-bbb") {
		t.Error("URL should contain state-bbb")
	}
}

func TestAuthorizeURL_SpecialCharsInState(t *testing.T) {
	tests := []struct {
		name  string
		state string
	}{
		{"spaces", "state with spaces"},
		{"ampersand", "state&value"},
		{"equals", "state=value"},
		{"plus", "state+value"},
		{"unicode", "stav-čeština"},
		{"url characters", "https://example.com?a=1&b=2"},
	}

	cfg := newTestConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawURL := cfg.AuthorizeURL(tt.state)

			parsed, err := url.Parse(rawURL)
			if err != nil {
				t.Fatalf("AuthorizeURL returned invalid URL for state %q: %v", tt.state, err)
			}

			got := parsed.Query().Get("state")
			if got != tt.state {
				t.Errorf("state round-trip failed: got %q, want %q", got, tt.state)
			}
		})
	}
}

func TestAuthorizeURL_DifferentConfigs(t *testing.T) {
	tests := []struct {
		name        string
		clientID    string
		redirectURI string
	}{
		{"default config", "my-app-id", "http://localhost:8090/api/reddit/callback"},
		{"custom domain", "prod-app", "https://myapp.com/auth/callback"},
		{"different port", "dev-app", "http://localhost:3000/callback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &OAuthConfig{
				ClientID:     tt.clientID,
				ClientSecret: "secret",
				RedirectURI:  tt.redirectURI,
				UserAgent:    "Test/1.0",
			}

			rawURL := cfg.AuthorizeURL("state")
			parsed, err := url.Parse(rawURL)
			if err != nil {
				t.Fatalf("invalid URL: %v", err)
			}

			params := parsed.Query()
			if got := params.Get("client_id"); got != tt.clientID {
				t.Errorf("client_id = %q, want %q", got, tt.clientID)
			}
			if got := params.Get("redirect_uri"); got != tt.redirectURI {
				t.Errorf("redirect_uri = %q, want %q", got, tt.redirectURI)
			}
		})
	}
}

func TestAuthorizeURL_NoExtraParams(t *testing.T) {
	cfg := newTestConfig()
	rawURL := cfg.AuthorizeURL("state")

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{
		"client_id":     true,
		"response_type": true,
		"state":         true,
		"redirect_uri":  true,
		"duration":      true,
		"scope":         true,
	}

	for key := range parsed.Query() {
		if !expected[key] {
			t.Errorf("unexpected query parameter: %q", key)
		}
	}

	if len(parsed.Query()) != len(expected) {
		t.Errorf("expected %d params, got %d", len(expected), len(parsed.Query()))
	}
}
