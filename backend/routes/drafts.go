package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"redveluvanto/ai"
	"redveluvanto/crypto"
	"redveluvanto/reddit"
)

func RegisterDraftRoutes(e *core.ServeEvent, aiClient *ai.Client, redditClient *reddit.Client, oauthConfig *reddit.OAuthConfig) {
	e.Router.POST("/api/drafts/generate", func(re *core.RequestEvent) error {
		var body struct {
			ThreadID        string `json:"thread_id"`
			PersonaID       string `json:"persona_id"`
			ParentCommentID string `json:"parent_comment_id"`
		}
		if err := json.NewDecoder(re.Request.Body).Decode(&body); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}

		thread, err := re.App.FindRecordById("threads", body.ThreadID)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "thread not found"})
		}

		if thread.GetString("comments_tree") == "" || thread.GetString("comments_tree") == "null" {
			if err := ensureComments(re, thread, redditClient, oauthConfig); err != nil {
				log.Printf("drafts: failed to fetch comments for thread %s: %v", body.ThreadID, err)
			}
		}

		persona, err := re.App.FindRecordById("personas", body.PersonaID)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "persona not found"})
		}

		generated, err := aiClient.GenerateReply(re.Request.Context(), persona, thread, body.ParentCommentID, ai.LoadProductContext(re.App), ai.LoadGlobalForbiddenWords(re.App))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		col, err := re.App.FindCollectionByNameOrId("drafts")
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": "drafts collection not found"})
		}

		detection, _ := aiClient.DetectAIContent(re.Request.Context(), generated)

		draft := core.NewRecord(col)
		draft.Set("thread", body.ThreadID)
		draft.Set("persona", body.PersonaID)
		draft.Set("user", re.Auth.Id)
		draft.Set("parent_comment_id", body.ParentCommentID)
		draft.Set("generated_text", generated)
		draft.Set("status", "draft")
		if detection != nil {
			draft.Set("ai_detection_score", detection.Score)
			flagsJSON, _ := json.Marshal(detection.Flags)
			draft.Set("ai_detection_flags", string(flagsJSON))
		}

		if err := re.App.Save(draft); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusOK, draft)
	}).Bind(apis.RequireAuth())

	e.Router.POST("/api/drafts/{id}/regenerate", func(re *core.RequestEvent) error {
		id := re.Request.PathValue("id")

		existing, err := re.App.FindRecordById("drafts", id)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "draft not found"})
		}
		if existing.GetString("user") != re.Auth.Id {
			return re.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		}

		thread, err := re.App.FindRecordById("threads", existing.GetString("thread"))
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "thread not found"})
		}

		if thread.GetString("comments_tree") == "" || thread.GetString("comments_tree") == "null" {
			if err := ensureComments(re, thread, redditClient, oauthConfig); err != nil {
				log.Printf("drafts: failed to fetch comments for thread %s: %v", existing.GetString("thread"), err)
			}
		}

		persona, err := re.App.FindRecordById("personas", existing.GetString("persona"))
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "persona not found"})
		}

		generated, err := aiClient.GenerateReply(re.Request.Context(), persona, thread, existing.GetString("parent_comment_id"), ai.LoadProductContext(re.App), ai.LoadGlobalForbiddenWords(re.App))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		col, err := re.App.FindCollectionByNameOrId("drafts")
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": "drafts collection not found"})
		}

		detection, _ := aiClient.DetectAIContent(re.Request.Context(), generated)

		newDraft := core.NewRecord(col)
		newDraft.Set("thread", existing.GetString("thread"))
		newDraft.Set("persona", existing.GetString("persona"))
		newDraft.Set("user", re.Auth.Id)
		newDraft.Set("parent_comment_id", existing.GetString("parent_comment_id"))
		newDraft.Set("generated_text", generated)
		newDraft.Set("status", "draft")
		if detection != nil {
			newDraft.Set("ai_detection_score", detection.Score)
			flagsJSON, _ := json.Marshal(detection.Flags)
			newDraft.Set("ai_detection_flags", string(flagsJSON))
		}

		if err := re.App.Save(newDraft); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusOK, newDraft)
	}).Bind(apis.RequireAuth())

	e.Router.PATCH("/api/drafts/{id}", func(re *core.RequestEvent) error {
		id := re.Request.PathValue("id")

		draft, err := re.App.FindRecordById("drafts", id)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "draft not found"})
		}
		if draft.GetString("user") != re.Auth.Id {
			return re.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		}

		var body struct {
			EditedText string `json:"edited_text"`
		}
		if err := json.NewDecoder(re.Request.Body).Decode(&body); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}

		draft.Set("edited_text", body.EditedText)
		if err := re.App.Save(draft); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusOK, draft)
	}).Bind(apis.RequireAuth())

	e.Router.POST("/api/drafts/{id}/approve", func(re *core.RequestEvent) error {
		id := re.Request.PathValue("id")

		draft, err := re.App.FindRecordById("drafts", id)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "draft not found"})
		}
		if draft.GetString("user") != re.Auth.Id {
			return re.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		}

		_, err = re.App.FindFirstRecordByFilter("reddit_accounts", "user = {:uid} && is_active = true", dbx.Params{"uid": re.Auth.Id})
		if err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "no active reddit account connected"})
		}

		draft.Set("status", "queued")
		draft.Set("queued_at", time.Now().UTC())
		if err := re.App.Save(draft); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusOK, draft)
	}).Bind(apis.RequireAuth())

	e.Router.GET("/api/drafts/queue-status", func(re *core.RequestEvent) error {
		threadID := re.Request.URL.Query().Get("thread_id")

		dailyLimit := 50
		limitRecord, err := re.App.FindFirstRecordByFilter("settings", "key = 'post_daily_limit'")
		if err == nil {
			if v, parseErr := strconv.Atoi(limitRecord.GetString("value")); parseErr == nil && v > 0 {
				dailyLimit = v
			}
		}

		todayStart := time.Now().UTC().Truncate(24 * time.Hour)
		todayPosts, _ := re.App.FindRecordsByFilter("drafts",
			"status = 'posted' && user = {:uid} && posted_at >= {:since}", "", 0, 0,
			dbx.Params{"uid": re.Auth.Id, "since": todayStart.Format(time.RFC3339)})

		queued, _ := re.App.FindRecordsByFilter("drafts",
			"status = 'queued' && user = {:uid}", "", 0, 0,
			dbx.Params{"uid": re.Auth.Id})

		isFollowUp := false
		if threadID != "" {
			existing, _ := re.App.FindFirstRecordByFilter("drafts",
				"status = 'posted' && user = {:uid} && thread = {:tid}",
				dbx.Params{"uid": re.Auth.Id, "tid": threadID})
			isFollowUp = existing != nil
		}

		canQueue := isFollowUp || len(todayPosts) < dailyLimit

		return re.JSON(http.StatusOK, map[string]interface{}{
			"posted_today": len(todayPosts),
			"daily_limit":  dailyLimit,
			"queued":       len(queued),
			"is_follow_up": isFollowUp,
			"can_queue":    canQueue,
		})
	}).Bind(apis.RequireAuth())

	e.Router.GET("/api/drafts/queue", func(re *core.RequestEvent) error {
		drafts, err := re.App.FindRecordsByFilter("drafts",
			"(status = 'queued' || status = 'posting') && user = {:uid}",
			"queued_at", 0, 50,
			dbx.Params{"uid": re.Auth.Id})
		if err != nil {
			return re.JSON(http.StatusOK, []interface{}{})
		}

		type queueItem struct {
			ID          string `json:"id"`
			ThreadID    string `json:"thread_id"`
			ThreadTitle string `json:"thread_title"`
			Subreddit   string `json:"subreddit"`
			Status      string `json:"status"`
			QueuedAt    string `json:"queued_at"`
			TextPreview string `json:"text_preview"`
		}

		items := make([]queueItem, 0, len(drafts))
		for _, d := range drafts {
			threadID := d.GetString("thread")
			var title, subreddit string
			if t, tErr := re.App.FindRecordById("threads", threadID); tErr == nil {
				title = t.GetString("title")
				subreddit = t.GetString("subreddit")
			}

			text := d.GetString("edited_text")
			if text == "" {
				text = d.GetString("generated_text")
			}
			if len(text) > 120 {
				text = text[:120] + "…"
			}

			items = append(items, queueItem{
				ID:          d.Id,
				ThreadID:    threadID,
				ThreadTitle: title,
				Subreddit:   subreddit,
				Status:      d.GetString("status"),
				QueuedAt:    d.GetDateTime("queued_at").Time().Format(time.RFC3339),
				TextPreview: text,
			})
		}

		return re.JSON(http.StatusOK, items)
	}).Bind(apis.RequireAuth())

	e.Router.POST("/api/drafts/{id}/cancel", func(re *core.RequestEvent) error {
		id := re.Request.PathValue("id")

		draft, err := re.App.FindRecordById("drafts", id)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "draft not found"})
		}
		if draft.GetString("user") != re.Auth.Id {
			return re.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		}
		if draft.GetString("status") != "queued" {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "only queued drafts can be cancelled"})
		}

		draft.Set("status", "draft")
		if err := re.App.Save(draft); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusOK, draft)
	}).Bind(apis.RequireAuth())
}

func ensureComments(re *core.RequestEvent, thread *core.Record, redditClient *reddit.Client, oauthConfig *reddit.OAuthConfig) error {
	account, err := re.App.FindFirstRecordByFilter("reddit_accounts", "user = {:uid} && is_active = true", dbx.Params{"uid": re.Auth.Id})
	if err != nil {
		return fmt.Errorf("no active reddit account: %w", err)
	}

	encKey := os.Getenv("TOKEN_ENCRYPTION_KEY")
	if encKey == "" {
		encKey = os.Getenv("LITELLM_MASTER_KEY")
	}
	if encKey == "" {
		encKey = "redveluvanto-default-key"
	}

	var accessToken string
	tokenExpiry := account.GetDateTime("token_expiry").Time()
	if time.Now().Add(5 * time.Minute).After(tokenExpiry) {
		rawRefresh := account.GetString("refresh_token")
		refreshToken, decErr := crypto.Decrypt(rawRefresh, encKey)
		if decErr != nil {
			refreshToken = rawRefresh
		}
		tokens, refErr := oauthConfig.RefreshToken(re.Request.Context(), refreshToken)
		if refErr != nil {
			return fmt.Errorf("failed to refresh token: %w", refErr)
		}
		encAccess, _ := crypto.Encrypt(tokens.AccessToken, encKey)
		account.Set("access_token", encAccess)
		account.Set("token_expiry", time.Now().Add(time.Duration(tokens.ExpiresIn)*time.Second))
		if saveErr := re.App.Save(account); saveErr != nil {
			return fmt.Errorf("failed to save token: %w", saveErr)
		}
		accessToken = tokens.AccessToken
	} else {
		rawAccess := account.GetString("access_token")
		decrypted, decErr := crypto.Decrypt(rawAccess, encKey)
		if decErr != nil {
			accessToken = rawAccess
		} else {
			accessToken = decrypted
		}
	}

	postWithComments, err := redditClient.GetPostComments(re.Request.Context(), accessToken, thread.GetString("reddit_id"))
	if err != nil {
		return fmt.Errorf("failed to fetch comments: %w", err)
	}

	commentsJSON, _ := json.Marshal(postWithComments.Comments)
	thread.Set("comments_tree", string(commentsJSON))
	thread.Set("num_comments", postWithComments.Post.NumComments)

	return re.App.Save(thread)
}
