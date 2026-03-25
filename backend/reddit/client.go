package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client struct {
	httpClient         *http.Client
	userAgent          string
	rateLimitRemaining int
	rateLimitReset     time.Time
	mu                 sync.Mutex
}

type Post struct {
	ID          string  `json:"id"`
	Subreddit   string  `json:"subreddit"`
	Title       string  `json:"title"`
	SelfText    string  `json:"selftext"`
	URL         string  `json:"url"`
	Permalink   string  `json:"permalink"`
	Author      string  `json:"author"`
	Score       int     `json:"score"`
	NumComments int     `json:"num_comments"`
	CreatedUTC  float64 `json:"created_utc"`
}

type Comment struct {
	ID         string     `json:"id"`
	Author     string     `json:"author"`
	Body       string     `json:"body"`
	Score      int        `json:"score"`
	CreatedUTC float64    `json:"created_utc"`
	ParentID   string     `json:"parent_id"`
	Depth      int        `json:"depth"`
	Replies    []*Comment `json:"replies"`
}

type PostWithComments struct {
	Post     Post       `json:"post"`
	Comments []*Comment `json:"comments"`
}

type SearchResult struct {
	Posts []Post `json:"posts"`
	After string `json:"after"`
}

type SubredditRule struct {
	ShortName   string `json:"short_name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type SubredditAbout struct {
	DisplayName       string `json:"display_name"`
	PublicDescription string `json:"public_description"`
	Description       string `json:"description"`
	Subscribers       int    `json:"subscribers"`
}

type CommentResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	Permalink string `json:"permalink"`
}

type RedditUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type redditListing struct {
	Kind string            `json:"kind"`
	Data redditListingData `json:"data"`
}

type redditListingData struct {
	After    string             `json:"after"`
	Children []redditThingChild `json:"children"`
}

type redditThingChild struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

type rawComment struct {
	ID         string          `json:"id"`
	Author     string          `json:"author"`
	Body       string          `json:"body"`
	Score      int             `json:"score"`
	CreatedUTC float64         `json:"created_utc"`
	ParentID   string          `json:"parent_id"`
	Depth      int             `json:"depth"`
	Replies    json.RawMessage `json:"replies"`
}

func NewClient(userAgent string) *Client {
	return &Client{
		httpClient:         &http.Client{Timeout: 30 * time.Second},
		userAgent:          userAgent,
		rateLimitRemaining: 60,
	}
}

func (c *Client) doRequest(ctx context.Context, method, rawURL, accessToken string, body io.Reader, contentType string) (*http.Response, error) {
	c.mu.Lock()
	if c.rateLimitRemaining <= 5 && time.Now().Before(c.rateLimitReset) {
		wait := time.Until(c.rateLimitReset)
		c.mu.Unlock()
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		c.mu.Unlock()
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	if accessToken != "" {
		req.Header.Set("Authorization", "bearer "+accessToken)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if remaining := resp.Header.Get("X-Ratelimit-Remaining"); remaining != "" {
		if val, parseErr := strconv.ParseFloat(remaining, 64); parseErr == nil {
			c.rateLimitRemaining = int(val)
		}
	}
	if reset := resp.Header.Get("X-Ratelimit-Reset"); reset != "" {
		if secs, parseErr := strconv.ParseInt(reset, 10, 64); parseErr == nil {
			c.rateLimitReset = time.Now().Add(time.Duration(secs) * time.Second)
		}
	}
	c.mu.Unlock()

	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()
		c.mu.Lock()
		wait := time.Until(c.rateLimitReset)
		if wait <= 0 {
			wait = 10 * time.Second
		}
		c.mu.Unlock()
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return c.doRequest(ctx, method, rawURL, accessToken, nil, contentType)
	}

	return resp, nil
}

func (c *Client) SearchPosts(ctx context.Context, accessToken, subreddit, query string, limit int, after string) (*SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("sort", "new")
	params.Set("limit", strconv.Itoa(limit))
	params.Set("raw_json", "1")
	params.Set("type", "link")
	if after != "" {
		params.Set("after", after)
	}

	var endpoint string
	if subreddit != "" {
		params.Set("restrict_sr", "true")
		endpoint = fmt.Sprintf("https://oauth.reddit.com/r/%s/search?%s", subreddit, params.Encode())
	} else {
		endpoint = "https://oauth.reddit.com/search?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, accessToken, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit search returned status %d", resp.StatusCode)
	}

	var listing redditListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}

	result := &SearchResult{After: listing.Data.After}
	for _, child := range listing.Data.Children {
		var post Post
		if err := json.Unmarshal(child.Data, &post); err != nil {
			continue
		}
		result.Posts = append(result.Posts, post)
	}
	return result, nil
}

func (c *Client) GetPostComments(ctx context.Context, accessToken, postID string) (*PostWithComments, error) {
	endpoint := fmt.Sprintf("https://oauth.reddit.com/comments/%s?raw_json=1&limit=500&sort=top", postID)
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, accessToken, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit get comments returned status %d", resp.StatusCode)
	}

	var listings []redditListing
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		return nil, fmt.Errorf("decoding comments response: %w", err)
	}

	if len(listings) < 2 {
		return nil, fmt.Errorf("unexpected response format: expected 2 listings, got %d", len(listings))
	}

	var post Post
	if len(listings[0].Data.Children) > 0 {
		if err := json.Unmarshal(listings[0].Data.Children[0].Data, &post); err != nil {
			return nil, fmt.Errorf("decoding post data: %w", err)
		}
	}

	comments := parseCommentChildren(listings[1].Data.Children)
	return &PostWithComments{Post: post, Comments: comments}, nil
}

func parseCommentChildren(children []redditThingChild) []*Comment {
	var comments []*Comment
	for _, child := range children {
		if child.Kind != "t1" {
			continue
		}
		var raw rawComment
		if err := json.Unmarshal(child.Data, &raw); err != nil {
			continue
		}
		comment := &Comment{
			ID:         raw.ID,
			Author:     raw.Author,
			Body:       raw.Body,
			Score:      raw.Score,
			CreatedUTC: raw.CreatedUTC,
			ParentID:   raw.ParentID,
			Depth:      raw.Depth,
		}
		// replies is either the empty string "" or a Listing object
		if len(raw.Replies) > 0 && raw.Replies[0] == '{' {
			var repliesListing redditListing
			if err := json.Unmarshal(raw.Replies, &repliesListing); err == nil {
				comment.Replies = parseCommentChildren(repliesListing.Data.Children)
			}
		}
		comments = append(comments, comment)
	}
	return comments
}

func (c *Client) GetSubredditRules(ctx context.Context, accessToken, subreddit string) ([]SubredditRule, error) {
	endpoint := fmt.Sprintf("https://oauth.reddit.com/r/%s/about/rules", subreddit)
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, accessToken, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit get rules returned status %d", resp.StatusCode)
	}

	var result struct {
		Rules []SubredditRule `json:"rules"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding rules response: %w", err)
	}
	return result.Rules, nil
}

func (c *Client) GetSubredditAbout(ctx context.Context, accessToken, subreddit string) (*SubredditAbout, error) {
	endpoint := fmt.Sprintf("https://oauth.reddit.com/r/%s/about", subreddit)
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, accessToken, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit get subreddit about returned status %d", resp.StatusCode)
	}

	var wrapper struct {
		Kind string         `json:"kind"`
		Data SubredditAbout `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decoding subreddit about response: %w", err)
	}
	return &wrapper.Data, nil
}

func (c *Client) SubmitComment(ctx context.Context, accessToken, parentFullname, text string) (*CommentResponse, error) {
	params := url.Values{}
	params.Set("api_type", "json")
	params.Set("thing_id", parentFullname)
	params.Set("text", text)
	params.Set("return_rtjson", "true")

	resp, err := c.doRequest(ctx, http.MethodPost, "https://oauth.reddit.com/api/comment",
		accessToken, strings.NewReader(params.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit submit comment returned status %d", resp.StatusCode)
	}

	var comment CommentResponse
	if err := json.NewDecoder(resp.Body).Decode(&comment); err != nil {
		return nil, fmt.Errorf("decoding comment response: %w", err)
	}
	return &comment, nil
}

func (c *Client) GetMe(ctx context.Context, accessToken string) (*RedditUser, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "https://oauth.reddit.com/api/v1/me", accessToken, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit get me returned status %d", resp.StatusCode)
	}

	var user RedditUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decoding user response: %w", err)
	}
	return &user, nil
}
