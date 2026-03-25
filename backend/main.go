package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	_ "redveluvanto/migrations"
	"redveluvanto/ai"
	"redveluvanto/reddit"
	"redveluvanto/routes"
	"redveluvanto/worker"
)

func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangGo,
		Automigrate:  true,
		Dir:          "migrations",
	})

	var monitor *worker.Monitor

	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		if monitor != nil {
			monitor.Stop()
		}
		return e.Next()
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		oauthConfig := &reddit.OAuthConfig{
			ClientID:     os.Getenv("REDDIT_CLIENT_ID"),
			ClientSecret: os.Getenv("REDDIT_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("REDDIT_REDIRECT_URI"),
			UserAgent:    "web:redveluvanto:v0.1.0 (by /u/redveluvanto)",
		}
		redditClient := reddit.NewClient(oauthConfig.UserAgent)

		litellmURL := os.Getenv("LITELLM_URL")
		if litellmURL == "" {
			litellmURL = "http://litellm:4000"
		}
		litellmKey := os.Getenv("LITELLM_MASTER_KEY")
		aiModel := os.Getenv("AI_FAST_MODEL")
		if aiModel == "" {
			aiModel = "gemini-2.5-flash"
		}
		aiClient := ai.NewClient(litellmURL, litellmKey, aiModel)

		routes.RegisterRedditRoutes(se, oauthConfig, redditClient)
		routes.RegisterDraftRoutes(se, aiClient, redditClient, oauthConfig)
		routes.RegisterPersonaRoutes(se, aiClient)
		routes.RegisterAdminRoutes(se, aiClient)
		routes.RegisterKnowledgeRoutes(se)
		routes.RegisterThreadRoutes(se, redditClient, oauthConfig)
		monitor = worker.NewMonitor(se.App, redditClient, oauthConfig, aiClient)
		monitor.Start()
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start PocketBase: %v", err)
	}
}
