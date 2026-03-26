package worker

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"redveluvanto/crypto"
	"redveluvanto/reddit"
)

type Poster struct {
	app         core.App
	client      *reddit.Client
	oauthConfig *reddit.OAuthConfig
	stopCh      chan struct{}
}

func NewPoster(app core.App, client *reddit.Client, oauthConfig *reddit.OAuthConfig) *Poster {
	return &Poster{
		app:         app,
		client:      client,
		oauthConfig: oauthConfig,
		stopCh:      make(chan struct{}),
	}
}

func (p *Poster) Start() {
	go func() {
		for {
			p.runCycle()

			select {
			case <-time.After(30 * time.Second):
			case <-p.stopCh:
				return
			}
		}
	}()
}

func (p *Poster) Stop() {
	close(p.stopCh)
}

func (p *Poster) getSettingInt(key string, fallback int) int {
	record, err := p.app.FindFirstRecordByFilter("settings", "key = {:k}", dbx.Params{"k": key})
	if err != nil {
		return fallback
	}
	val, err := strconv.Atoi(record.GetString("value"))
	if err != nil || val <= 0 {
		return fallback
	}
	return val
}

func (p *Poster) runCycle() {
	queued, err := p.app.FindRecordsByFilter("drafts", "status = 'queued'", "queued_at", 0, 50)
	if err != nil || len(queued) == 0 {
		return
	}

	globalDelay := time.Duration(p.getSettingInt("post_delay_global", 90)) * time.Second
	accountDelay := time.Duration(p.getSettingInt("post_delay_account", 120)) * time.Second
	subredditDelay := time.Duration(p.getSettingInt("post_delay_subreddit", 600)) * time.Second
	dailyLimit := p.getSettingInt("post_daily_limit", 50)

	for _, draft := range queued {
		if p.tryPost(draft, globalDelay, accountDelay, subredditDelay, dailyLimit) {
			return
		}
	}
}

func (p *Poster) tryPost(draft *core.Record, globalDelay, accountDelay, subredditDelay time.Duration, dailyLimit int) bool {
	userID := draft.GetString("user")
	threadID := draft.GetString("thread")

	thread, err := p.app.FindRecordById("threads", threadID)
	if err != nil {
		log.Printf("worker: poster: thread %s not found, marking failed", threadID)
		draft.Set("status", "failed")
		_ = p.app.Save(draft)
		return false
	}

	subreddit := thread.GetString("subreddit")

	redditCreatedAt := thread.GetDateTime("reddit_created_at").Time()
	if !redditCreatedAt.IsZero() && time.Since(redditCreatedAt) > 24*time.Hour {
		log.Printf("worker: poster: thread %s is older than 24h, marking draft %s as failed", threadID, draft.Id)
		draft.Set("status", "failed")
		_ = p.app.Save(draft)
		return false
	}

	lastGlobals, _ := p.app.FindRecordsByFilter("drafts", "status = 'posted'", "-posted_at", 0, 1)
	if len(lastGlobals) > 0 {
		elapsed := time.Since(lastGlobals[0].GetDateTime("posted_at").Time())
		if elapsed < globalDelay {
			return false
		}
	}

	existingInThread, _ := p.app.FindFirstRecordByFilter("drafts",
		"status = 'posted' && user = {:uid} && thread = {:tid}",
		dbx.Params{"uid": userID, "tid": threadID})
	isFollowUp := existingInThread != nil

	if !isFollowUp {
		lastAccounts, _ := p.app.FindRecordsByFilter("drafts",
			"status = 'posted' && user = {:uid}", "-posted_at", 0, 1,
			dbx.Params{"uid": userID})
		var lastAccount *core.Record
		if len(lastAccounts) > 0 {
			lastAccount = lastAccounts[0]
		}
		if lastAccount != nil {
			elapsed := time.Since(lastAccount.GetDateTime("posted_at").Time())
			if elapsed < accountDelay {
				return false
			}
		}

		lastSubreddit, _ := p.app.FindRecordsByFilter("drafts", "status = 'posted'", "-posted_at", 0, 10)
		for _, d := range lastSubreddit {
			t, tErr := p.app.FindRecordById("threads", d.GetString("thread"))
			if tErr != nil {
				continue
			}
			if t.GetString("subreddit") == subreddit {
				elapsed := time.Since(d.GetDateTime("posted_at").Time())
				if elapsed < subredditDelay {
					return false
				}
				break
			}
		}

		todayStart := time.Now().UTC().Truncate(24 * time.Hour)
		todayPosts, _ := p.app.FindRecordsByFilter("drafts",
			"status = 'posted' && user = {:uid} && posted_at >= {:since}", "", 0, 0,
			dbx.Params{"uid": userID, "since": todayStart.Format(time.RFC3339)})
		if len(todayPosts) >= dailyLimit {
			return false
		}
	}

	account, err := p.app.FindFirstRecordByFilter("reddit_accounts",
		"user = {:uid} && is_active = true", dbx.Params{"uid": userID})
	if err != nil {
		log.Printf("worker: poster: no reddit account for user %s, marking failed", userID)
		draft.Set("status", "failed")
		_ = p.app.Save(draft)
		return false
	}

	token, err := p.getTokenForAccount(account)
	if err != nil {
		log.Printf("worker: poster: token refresh failed for user %s: %v", userID, err)
		draft.Set("status", "failed")
		_ = p.app.Save(draft)
		return false
	}

	var parentFullname string
	if parentID := draft.GetString("parent_comment_id"); parentID != "" {
		if strings.HasPrefix(parentID, "t1_") || strings.HasPrefix(parentID, "t3_") {
			parentFullname = parentID
		} else {
			parentFullname = "t1_" + parentID
		}
	} else {
		parentFullname = "t3_" + thread.GetString("reddit_id")
	}

	text := draft.GetString("edited_text")
	if text == "" {
		text = draft.GetString("generated_text")
	}

	draft.Set("status", "posting")
	_ = p.app.Save(draft)

	ctx := context.Background()
	comment, err := p.client.SubmitComment(ctx, token, parentFullname, text)
	if err != nil {
		log.Printf("worker: poster: SubmitComment failed for draft %s: %v", draft.Id, err)
		draft.Set("status", "failed")
		_ = p.app.Save(draft)
		return true
	}

	draft.Set("status", "posted")
	draft.Set("posted_at", time.Now().UTC())
	draft.Set("reddit_comment_id", comment.ID)
	if err := p.app.Save(draft); err != nil {
		log.Printf("worker: poster: failed to save posted draft %s: %v", draft.Id, err)
	}

	status, err := p.app.FindFirstRecordByFilter("thread_status",
		"thread = {:tid} && user = {:uid}",
		dbx.Params{"tid": threadID, "uid": userID})
	if err == nil {
		status.Set("status", "replied")
		_ = p.app.Save(status)
	}

	log.Printf("worker: poster: posted draft %s to r/%s", draft.Id, subreddit)
	return true
}

func (p *Poster) getTokenForAccount(account *core.Record) (string, error) {
	encKey := os.Getenv("TOKEN_ENCRYPTION_KEY")
	if encKey == "" {
		encKey = os.Getenv("LITELLM_MASTER_KEY")
	}
	if encKey == "" {
		encKey = "redveluvanto-default-key"
	}

	tokenExpiry := account.GetDateTime("token_expiry").Time()
	if time.Now().Add(5 * time.Minute).After(tokenExpiry) {
		rawRefresh := account.GetString("refresh_token")
		refreshToken, err := crypto.Decrypt(rawRefresh, encKey)
		if err != nil {
			refreshToken = rawRefresh
		}

		ctx := context.Background()
		tokens, err := p.oauthConfig.RefreshToken(ctx, refreshToken)
		if err != nil {
			return "", err
		}

		encAccess, _ := crypto.Encrypt(tokens.AccessToken, encKey)
		account.Set("access_token", encAccess)
		account.Set("token_expiry", time.Now().Add(time.Duration(tokens.ExpiresIn)*time.Second))
		if err := p.app.Save(account); err != nil {
			return "", err
		}
		return tokens.AccessToken, nil
	}

	rawAccess := account.GetString("access_token")
	accessToken, err := crypto.Decrypt(rawAccess, encKey)
	if err != nil {
		return rawAccess, nil
	}
	return accessToken, nil
}
