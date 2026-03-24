package reddit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	UserAgent    string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (c *OAuthConfig) AuthorizeURL(state string) string {
	params := url.Values{}
	params.Set("client_id", c.ClientID)
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("redirect_uri", c.RedirectURI)
	params.Set("duration", "permanent")
	params.Set("scope", "identity read submit")
	return "https://www.reddit.com/api/v1/authorize?" + params.Encode()
}

func (c *OAuthConfig) basicAuth() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(c.ClientID+":"+c.ClientSecret))
}

func (c *OAuthConfig) tokenRequest(ctx context.Context, params url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.reddit.com/api/v1/access_token", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", c.basicAuth())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit token request returned status %d", resp.StatusCode)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	return &token, nil
}

func (c *OAuthConfig) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", c.RedirectURI)
	return c.tokenRequest(ctx, params)
}

func (c *OAuthConfig) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)
	return c.tokenRequest(ctx, params)
}

func (c *OAuthConfig) RevokeToken(ctx context.Context, token string) error {
	params := url.Values{}
	params.Set("token", token)
	params.Set("token_type_hint", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.reddit.com/api/v1/revoke_token", strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", c.basicAuth())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("reddit revoke token returned status %d", resp.StatusCode)
	}
	return nil
}
