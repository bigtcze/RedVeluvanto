package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
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

		persona, err := re.App.FindRecordById("personas", body.PersonaID)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "persona not found"})
		}

		generated, err := aiClient.GenerateReply(re.Request.Context(), persona, thread, body.ParentCommentID)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		col, err := re.App.FindCollectionByNameOrId("drafts")
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": "drafts collection not found"})
		}

		draft := core.NewRecord(col)
		draft.Set("thread", body.ThreadID)
		draft.Set("persona", body.PersonaID)
		draft.Set("user", re.Auth.Id)
		draft.Set("parent_comment_id", body.ParentCommentID)
		draft.Set("generated_text", generated)
		draft.Set("status", "draft")

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

		persona, err := re.App.FindRecordById("personas", existing.GetString("persona"))
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "persona not found"})
		}

		generated, err := aiClient.GenerateReply(re.Request.Context(), persona, thread, existing.GetString("parent_comment_id"))
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		col, err := re.App.FindCollectionByNameOrId("drafts")
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": "drafts collection not found"})
		}

		newDraft := core.NewRecord(col)
		newDraft.Set("thread", existing.GetString("thread"))
		newDraft.Set("persona", existing.GetString("persona"))
		newDraft.Set("user", re.Auth.Id)
		newDraft.Set("parent_comment_id", existing.GetString("parent_comment_id"))
		newDraft.Set("generated_text", generated)
		newDraft.Set("status", "draft")

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

		account, err := re.App.FindFirstRecordByFilter("reddit_accounts", "user = {:uid} && is_active = true", dbx.Params{"uid": re.Auth.Id})
		if err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "no active reddit account connected"})
		}

		encKey := os.Getenv("TOKEN_ENCRYPTION_KEY")
		if encKey == "" {
			encKey = os.Getenv("LITELLM_MASTER_KEY")
		}
		if encKey == "" {
			encKey = "redveluvanto-default-key"
		}

		var redditToken string
		tokenExpiry := account.GetDateTime("token_expiry").Time()
		if time.Now().Add(5 * time.Minute).After(tokenExpiry) {
			rawRefresh := account.GetString("refresh_token")
			refreshToken, err := crypto.Decrypt(rawRefresh, encKey)
			if err != nil {
				refreshToken = rawRefresh
			}
			tokens, err := oauthConfig.RefreshToken(re.Request.Context(), refreshToken)
			if err != nil {
				return re.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to refresh reddit token"})
			}
			encAccess, _ := crypto.Encrypt(tokens.AccessToken, encKey)
			account.Set("access_token", encAccess)
			account.Set("token_expiry", time.Now().Add(time.Duration(tokens.ExpiresIn)*time.Second))
			if err := re.App.Save(account); err != nil {
				return re.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save refreshed token"})
			}
			redditToken = tokens.AccessToken
		} else {
			rawAccess := account.GetString("access_token")
			decrypted, err := crypto.Decrypt(rawAccess, encKey)
			if err != nil {
				redditToken = rawAccess
			} else {
				redditToken = decrypted
			}
		}

		thread, err := re.App.FindRecordById("threads", draft.GetString("thread"))
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "thread not found"})
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

		comment, err := redditClient.SubmitComment(re.Request.Context(), redditToken, parentFullname, text)
		if err != nil {
			draft.Set("status", "failed")
			_ = re.App.Save(draft)
			return re.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}

		draft.Set("status", "posted")
		draft.Set("posted_at", time.Now().UTC())
		draft.Set("reddit_comment_id", comment.ID)
		if err := re.App.Save(draft); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		status, err := re.App.FindFirstRecordByFilter("thread_status",
			"thread = {:tid} && user = {:uid}",
			dbx.Params{"tid": draft.GetString("thread"), "uid": re.Auth.Id})
		if err == nil {
			status.Set("status", "replied")
			_ = re.App.Save(status)
		}

		return re.JSON(http.StatusOK, draft)
	}).Bind(apis.RequireAuth())
}
