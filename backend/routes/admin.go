package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"redveluvanto/ai"
)

func isSuperuser(app core.App, authRecord *core.Record) bool {
	if authRecord == nil {
		return false
	}
	firstUsers, err := app.FindRecordsByFilter("users", "", "created", 1, 0)
	if err != nil || len(firstUsers) == 0 {
		return false
	}
	return firstUsers[0].Id == authRecord.Id || firstUsers[0].Email() == authRecord.Email()
}

func RegisterAdminRoutes(e *core.ServeEvent, aiClient *ai.Client) {
	e.Router.GET("/api/setup/status", func(re *core.RequestEvent) error {
		users, err := re.App.FindRecordsByFilter("users", "", "", 1, 0)
		needsSetup := err != nil || len(users) == 0
		return re.JSON(http.StatusOK, map[string]bool{"needsSetup": needsSetup})
	})

	e.Router.POST("/api/setup/init", func(re *core.RequestEvent) error {
		existing, _ := re.App.FindRecordsByFilter("users", "", "", 1, 0)
		if len(existing) > 0 {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "Already set up"})
		}

		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := re.BindBody(&body); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		}

		users, err := re.App.FindCollectionByNameOrId("users")
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": "Collection not found"})
		}
		user := core.NewRecord(users)
		user.SetEmail(body.Email)
		user.SetPassword(body.Password)
		user.Set("verified", true)
		if err := re.App.Save(user); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusOK, map[string]bool{"ok": true})
	})

	e.Router.GET("/api/admin/debug", func(re *core.RequestEvent) error {
		return re.JSON(http.StatusOK, map[string]interface{}{
			"auth_id":         re.Auth.Id,
			"auth_email":      re.Auth.Email(),
			"auth_collection": re.Auth.Collection().Name,
			"is_superuser":    isSuperuser(re.App, re.Auth),
		})
	}).Bind(apis.RequireAuth())

	e.Router.GET("/api/admin/users", func(re *core.RequestEvent) error {
		if !isSuperuser(re.App, re.Auth) {
			return re.ForbiddenError("Admin access required", nil)
		}

		records, err := re.App.FindRecordsByFilter("users", "", "-created", 0, 10000)
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusOK, records)
	}).Bind(apis.RequireAuth())

	e.Router.POST("/api/admin/users", func(re *core.RequestEvent) error {
		if !isSuperuser(re.App, re.Auth) {
			return re.ForbiddenError("Admin access required", nil)
		}

		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := re.BindBody(&body); err != nil {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		}

		users, err := re.App.FindCollectionByNameOrId("users")
		if err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": "Collection not found"})
		}
		record := core.NewRecord(users)
		record.SetEmail(body.Email)
		record.SetPassword(body.Password)
		record.Set("verified", true)
		if err := re.App.Save(record); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.JSON(http.StatusCreated, record)
	}).Bind(apis.RequireAuth())

	e.Router.DELETE("/api/admin/users/{id}", func(re *core.RequestEvent) error {
		if !isSuperuser(re.App, re.Auth) {
			return re.ForbiddenError("Admin access required", nil)
		}

		id := re.Request.PathValue("id")
		if re.Auth.Id == id {
			return re.JSON(http.StatusBadRequest, map[string]string{"error": "Cannot delete yourself"})
		}

		record, err := re.App.FindRecordById("users", id)
		if err != nil {
			return re.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}

		if err := re.App.Delete(record); err != nil {
			return re.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return re.NoContent(http.StatusNoContent)
	}).Bind(apis.RequireAuth())

	e.Router.GET("/api/ai/models", func(re *core.RequestEvent) error {
		models, err := aiClient.ListModels(re.Request.Context())
		if err != nil {
			return re.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		return re.JSON(http.StatusOK, models)
	}).Bind(apis.RequireAuth())
}
