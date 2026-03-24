package worker

import (
	"context"
	"os"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"redveluvanto/crypto"
	"redveluvanto/reddit"
)

func GetValidToken(app core.App, oauthConfig *reddit.OAuthConfig) (string, error) {
	record, err := app.FindFirstRecordByFilter("reddit_accounts", "is_active = true")
	if err != nil {
		return "", err
	}

	encKey := os.Getenv("TOKEN_ENCRYPTION_KEY")
	if encKey == "" {
		encKey = os.Getenv("LITELLM_MASTER_KEY")
	}
	if encKey == "" {
		encKey = "redveluvanto-default-key"
	}

	tokenExpiry := record.GetDateTime("token_expiry").Time()
	if time.Now().Add(5 * time.Minute).After(tokenExpiry) {
		rawRefresh := record.GetString("refresh_token")
		refreshToken, err := crypto.Decrypt(rawRefresh, encKey)
		if err != nil {
			refreshToken = rawRefresh
		}

		ctx := context.Background()
		tokens, err := oauthConfig.RefreshToken(ctx, refreshToken)
		if err != nil {
			return "", err
		}

		encAccess, _ := crypto.Encrypt(tokens.AccessToken, encKey)
		record.Set("access_token", encAccess)
		record.Set("token_expiry", time.Now().Add(time.Duration(tokens.ExpiresIn)*time.Second))
		if err := app.Save(record); err != nil {
			return "", err
		}
		return tokens.AccessToken, nil
	}

	rawAccess := record.GetString("access_token")
	accessToken, err := crypto.Decrypt(rawAccess, encKey)
	if err != nil {
		return rawAccess, nil
	}
	return accessToken, nil
}
