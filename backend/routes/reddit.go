package routes

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"redveluvanto/crypto"
	"redveluvanto/reddit"
)

type stateEntry struct {
	userID    string
	createdAt time.Time
}

var (
	stateStore sync.Map
	stateTTL   = 10 * time.Minute
)

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func cleanExpiredStates() {
	now := time.Now()
	stateStore.Range(func(k, v any) bool {
		if entry, ok := v.(stateEntry); ok && now.Sub(entry.createdAt) > stateTTL {
			stateStore.Delete(k)
		}
		return true
	})
}

func RegisterRedditRoutes(e *core.ServeEvent, oauthConfig *reddit.OAuthConfig, client *reddit.Client) {
	e.Router.GET("/api/reddit/auth", func(re *core.RequestEvent) error {
		cleanExpiredStates()
		state, err := generateState()
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate state"})
		}
		stateStore.Store(state, stateEntry{userID: re.Auth.Id, createdAt: time.Now()})
		return re.Redirect(http.StatusFound, oauthConfig.AuthorizeURL(state))
	}).Bind(apis.RequireAuth())

	e.Router.GET("/api/reddit/callback", func(re *core.RequestEvent) error {
		q := re.Request.URL.Query()

		if errParam := q.Get("error"); errParam != "" {
			return re.Redirect(http.StatusFound, "/?reddit=error&reason="+errParam)
		}

		state := q.Get("state")
		code := q.Get("code")

		val, ok := stateStore.LoadAndDelete(state)
		if !ok {
			return re.Redirect(http.StatusFound, "/?reddit=error&reason=invalid_state")
		}
		entry := val.(stateEntry)

		tokens, err := oauthConfig.ExchangeCode(re.Request.Context(), code)
		if err != nil {
			return re.Redirect(http.StatusFound, "/?reddit=error&reason=exchange_failed")
		}

		user, err := client.GetMe(re.Request.Context(), tokens.AccessToken)
		if err != nil {
			return re.Redirect(http.StatusFound, "/?reddit=error&reason=identity_failed")
		}

		record, findErr := re.App.FindFirstRecordByFilter("reddit_accounts", "user = {:uid}", dbx.Params{"uid": entry.userID})
		if findErr != nil {
			col, err := re.App.FindCollectionByNameOrId("reddit_accounts")
			if err != nil {
				return re.Redirect(http.StatusFound, "/?reddit=error&reason=db_error")
			}
			record = core.NewRecord(col)
			record.Set("user", entry.userID)
		}

		encKey := os.Getenv("TOKEN_ENCRYPTION_KEY")
		if encKey == "" {
			encKey = os.Getenv("LITELLM_MASTER_KEY")
		}
		if encKey == "" {
			encKey = "redveluvanto-default-key"
		}
		encAccess, _ := crypto.Encrypt(tokens.AccessToken, encKey)
		encRefresh, _ := crypto.Encrypt(tokens.RefreshToken, encKey)

		record.Set("reddit_username", user.Name)
		record.Set("access_token", encAccess)
		record.Set("refresh_token", encRefresh)
		record.Set("token_expiry", time.Now().Add(time.Duration(tokens.ExpiresIn)*time.Second))
		record.Set("scopes", strings.Split(tokens.Scope, " "))
		record.Set("is_active", true)

		if err := re.App.Save(record); err != nil {
			return re.Redirect(http.StatusFound, "/?reddit=error&reason=save_failed")
		}

		return re.Redirect(http.StatusFound, "/?reddit=connected")
	})

	e.Router.GET("/api/reddit/status", func(re *core.RequestEvent) error {
		record, err := re.App.FindFirstRecordByFilter("reddit_accounts", "user = {:uid}", dbx.Params{"uid": re.Auth.Id})
		if err != nil {
			return re.JSON(http.StatusOK, map[string]any{
				"connected": false,
				"username":  "",
				"is_active": false,
			})
		}
		return re.JSON(http.StatusOK, map[string]any{
			"connected": true,
			"username":  record.GetString("reddit_username"),
			"is_active": record.GetBool("is_active"),
		})
	}).Bind(apis.RequireAuth())

	e.Router.POST("/api/reddit/disconnect", func(re *core.RequestEvent) error {
		record, err := re.App.FindFirstRecordByFilter("reddit_accounts", "user = {:uid}", dbx.Params{"uid": re.Auth.Id})
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "not connected"})
		}

		if raw := record.GetString("refresh_token"); raw != "" {
			encKey := os.Getenv("TOKEN_ENCRYPTION_KEY")
			if encKey == "" {
				encKey = os.Getenv("LITELLM_MASTER_KEY")
			}
			if encKey == "" {
				encKey = "redveluvanto-default-key"
			}
			refreshToken, err := crypto.Decrypt(raw, encKey)
			if err != nil {
				refreshToken = raw
			}
			_ = oauthConfig.RevokeToken(re.Request.Context(), refreshToken)
		}

		if err := re.App.Delete(record); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.NoContent(http.StatusNoContent)
	}).Bind(apis.RequireAuth())
}
