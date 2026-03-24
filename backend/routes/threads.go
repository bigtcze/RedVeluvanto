package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"redveluvanto/crypto"
	"redveluvanto/reddit"
)

func RegisterThreadRoutes(e *core.ServeEvent, redditClient *reddit.Client, oauthConfig *reddit.OAuthConfig) {
	e.Router.POST("/api/threads/{id}/refresh", func(re *core.RequestEvent) error {
		id := re.Request.PathValue("id")

		thread, err := re.App.FindRecordById("threads", id)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "thread not found"})
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

		var accessToken string
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
				return re.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save token"})
			}
			accessToken = tokens.AccessToken
		} else {
			rawAccess := account.GetString("access_token")
			decrypted, err := crypto.Decrypt(rawAccess, encKey)
			if err != nil {
				accessToken = rawAccess
			} else {
				accessToken = decrypted
			}
		}

		postWithComments, err := redditClient.GetPostComments(re.Request.Context(), accessToken, thread.GetString("reddit_id"))
		if err != nil {
			return re.JSON(http.StatusBadGateway, map[string]string{"error": "failed to fetch comments: " + err.Error()})
		}

		commentsJSON, _ := json.Marshal(postWithComments.Comments)
		thread.Set("comments_tree", string(commentsJSON))
		thread.Set("num_comments", postWithComments.Post.NumComments)

		if err := re.App.Save(thread); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusOK, thread)
	}).Bind(apis.RequireAuth())
}
