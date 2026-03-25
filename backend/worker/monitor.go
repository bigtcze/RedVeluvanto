package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"redveluvanto/ai"
	"redveluvanto/reddit"
)

type Monitor struct {
	app         core.App
	client      *reddit.Client
	oauthConfig *reddit.OAuthConfig
	aiClient    *ai.Client
	stopCh      chan struct{}
}

func NewMonitor(app core.App, client *reddit.Client, oauthConfig *reddit.OAuthConfig, aiClient *ai.Client) *Monitor {
	return &Monitor{
		app:         app,
		client:      client,
		oauthConfig: oauthConfig,
		aiClient:    aiClient,
		stopCh:      make(chan struct{}),
	}
}

func (m *Monitor) Start() {
	go func() {
		for {
			m.runCycle()

			interval := m.getInterval()
			select {
			case <-time.After(interval):
			case <-m.stopCh:
				return
			}
		}
	}()
}

func (m *Monitor) Stop() {
	close(m.stopCh)
}

func (m *Monitor) getSettingInt(key string, fallback int) int {
	record, err := m.app.FindFirstRecordByFilter("settings", "key = {:k}", dbx.Params{"k": key})
	if err != nil {
		return fallback
	}
	val, err := strconv.Atoi(record.GetString("value"))
	if err != nil || val <= 0 {
		return fallback
	}
	return val
}

func (m *Monitor) getInterval() time.Duration {
	record, err := m.app.FindFirstRecordByFilter("settings", "key = 'monitoring_interval'")
	if err != nil {
		return 5 * time.Minute
	}
	minutes, err := strconv.Atoi(record.GetString("value"))
	if err != nil || minutes <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(minutes) * time.Minute
}

func (m *Monitor) getMinScore() int {
	record, err := m.app.FindFirstRecordByFilter("settings", "key = 'min_relevance_score'")
	if err != nil {
		return 50
	}
	score, err := strconv.Atoi(record.GetString("value"))
	if err != nil || score <= 0 {
		return 50
	}
	return score
}

func (m *Monitor) runCycle() {
	ctx := context.Background()

	token, err := GetValidToken(m.app, m.oauthConfig)
	if err != nil {
		log.Printf("worker: monitor: no valid token: %v", err)
		return
	}

	keywords, err := m.app.FindRecordsByFilter("keywords", "is_active = true", "", 0, 0)
	if err != nil {
		log.Printf("worker: monitor: failed to find keywords: %v", err)
		return
	}

	for _, kw := range keywords {
		m.processKeyword(ctx, token, kw)
	}
}

type subredditMeta struct {
	rules []reddit.SubredditRule
	about *reddit.SubredditAbout
}

func (m *Monitor) processKeyword(ctx context.Context, token string, kw *core.Record) {
	query := kw.GetString("keyword")
	subreddits := kw.GetStringSlice("subreddits")

	metaCache := make(map[string]*subredditMeta)

	if len(subreddits) == 0 {
		m.searchAndProcess(ctx, token, kw, query, "", metaCache)
	} else {
		for _, sub := range subreddits {
			m.searchAndProcess(ctx, token, kw, query, sub, metaCache)
		}
	}
}

func (m *Monitor) searchAndProcess(ctx context.Context, token string, kw *core.Record, query, subreddit string, metaCache map[string]*subredditMeta) {
	results, err := m.client.SearchPosts(ctx, token, subreddit, query, 25, "")
	if err != nil {
		log.Printf("worker: monitor: search failed for %q in r/%s: %v", query, subreddit, err)
		return
	}

	for _, post := range results.Posts {
		if err := m.processPost(ctx, token, kw, post, metaCache); err != nil {
			log.Printf("worker: monitor: failed to process post %s: %v", post.ID, err)
		}
	}
}

func (m *Monitor) getSubredditMeta(ctx context.Context, token, subreddit string, metaCache map[string]*subredditMeta) *subredditMeta {
	if cached, ok := metaCache[subreddit]; ok {
		return cached
	}

	meta := &subredditMeta{}

	rules, err := m.client.GetSubredditRules(ctx, token, subreddit)
	if err != nil {
		log.Printf("worker: monitor: failed to get rules for r/%s: %v", subreddit, err)
	} else {
		meta.rules = rules
	}

	about, err := m.client.GetSubredditAbout(ctx, token, subreddit)
	if err != nil {
		log.Printf("worker: monitor: failed to get about for r/%s: %v", subreddit, err)
		about = &reddit.SubredditAbout{}
	}
	meta.about = about

	metaCache[subreddit] = meta
	return meta
}

func (m *Monitor) processPost(ctx context.Context, token string, kw *core.Record, post reddit.Post, metaCache map[string]*subredditMeta) error {
	maxAgeHours := m.getSettingInt("max_thread_age_hours", 12)
	postTime := time.Unix(int64(post.CreatedUTC), 0)
	if time.Since(postTime) > time.Duration(maxAgeHours)*time.Hour {
		return nil
	}

	_, err := m.app.FindFirstRecordByFilter("threads", "reddit_id = {:rid}", dbx.Params{"rid": post.ID})
	if err == nil {
		return nil
	}

	meta := m.getSubredditMeta(ctx, token, post.Subreddit, metaCache)
	rulesJSON, _ := json.Marshal(meta.rules)

	col, err := m.app.FindCollectionByNameOrId("threads")
	if err != nil {
		return fmt.Errorf("finding threads collection: %w", err)
	}

	record := core.NewRecord(col)
	record.Set("reddit_id", post.ID)
	record.Set("subreddit", post.Subreddit)
	record.Set("title", post.Title)
	record.Set("body", post.SelfText)
	record.Set("url", "https://reddit.com"+post.Permalink)
	record.Set("author", post.Author)
	record.Set("score", post.Score)
	record.Set("num_comments", post.NumComments)
	record.Set("subreddit_rules", string(rulesJSON))
	record.Set("subreddit_description", meta.about.Description)
	record.Set("matched_keyword", kw.Id)
	record.Set("found_at", time.Now().UTC())
	record.Set("reddit_created_at", postTime.UTC())

	if err := m.app.Save(record); err != nil {
		return fmt.Errorf("saving thread: %w", err)
	}

	var scoring *ai.ScoringResult
	if m.aiClient != nil {
		var scoreErr error
		scoring, scoreErr = m.aiClient.ScoreThread(ctx, post.Title, post.SelfText, post.Subreddit, meta.about.Description, kw.GetString("keyword"), ai.LoadProductContext(m.app))
		if scoreErr != nil {
			log.Printf("worker: monitor: AI scoring failed for %s: %v", post.ID, scoreErr)
		} else {
			record.Set("relevance_score", scoring.Score)
			record.Set("relevance_reason", scoring.Reason)
			if err := m.app.Save(record); err != nil {
				log.Printf("worker: monitor: failed to save scoring for %s: %v", post.ID, err)
			}
		}
	}

	users, err := m.app.FindRecordsByFilter("users", "id != ''", "", 0, 0)
	if err != nil {
		log.Printf("worker: monitor: failed to find users: %v", err)
	} else {
		statusCol, err := m.app.FindCollectionByNameOrId("thread_status")
		if err != nil {
			log.Printf("worker: monitor: failed to find thread_status collection: %v", err)
		} else {
			for _, user := range users {
				status := core.NewRecord(statusCol)
				status.Set("thread", record.Id)
				status.Set("user", user.Id)
				status.Set("status", "new")
				if err := m.app.Save(status); err != nil {
					log.Printf("worker: monitor: failed to save thread_status: %v", err)
				}
			}
		}
	}

	minScore := m.getMinScore()
	if scoring != nil && scoring.Score >= minScore {
		NotifyUsers(m.app, record, kw.GetString("keyword"))
	}

	return nil
}
