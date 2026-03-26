package routes

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

type draftTestCollections struct {
	users  *core.Collection
	drafts *core.Collection
}

func setupDraftTestCollections(t testing.TB, app *tests.TestApp) draftTestCollections {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		users = core.NewAuthCollection("users")
		if err := app.Save(users); err != nil {
			t.Fatalf("failed to create users collection: %v", err)
		}
	}

	drafts := core.NewBaseCollection("drafts")
	drafts.Fields.Add(&core.TextField{Name: "status"})
	drafts.Fields.Add(&core.TextField{Name: "user"})
	drafts.Fields.Add(&core.TextField{Name: "thread"})
	drafts.Fields.Add(&core.TextField{Name: "generated_text"})
	drafts.Fields.Add(&core.TextField{Name: "edited_text"})
	drafts.Fields.Add(&core.TextField{Name: "parent_comment_id"})
	drafts.Fields.Add(&core.TextField{Name: "persona"})
	drafts.Fields.Add(&core.TextField{Name: "reddit_comment_id"})
	drafts.Fields.Add(&core.DateField{Name: "posted_at"})
	drafts.Fields.Add(&core.TextField{Name: "queued_at"})
	if err := app.Save(drafts); err != nil {
		t.Fatalf("failed to create drafts collection: %v", err)
	}

	return draftTestCollections{users: users, drafts: drafts}
}

func draftTestAppFactory(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func createTestUser(t testing.TB, app *tests.TestApp, col *core.Collection, email string) (*core.Record, string) {
	user := core.NewRecord(col)
	user.SetEmail(email)
	user.SetPassword("password123456")
	user.Set("verified", true)
	if err := app.Save(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatalf("failed to generate auth token: %v", err)
	}
	return user, token
}

func createTestDraft(t testing.TB, app *tests.TestApp, col *core.Collection, userID, status string) *core.Record {
	draft := core.NewRecord(col)
	draft.Set("user", userID)
	draft.Set("status", status)
	draft.Set("generated_text", "AI generated reply")
	draft.Set("thread", "some_thread_id")
	if err := app.Save(draft); err != nil {
		t.Fatalf("failed to create draft: %v", err)
	}
	return draft
}

func TestCancelDraft_NotFound(t *testing.T) {
	var s tests.ApiScenario
	s = tests.ApiScenario{
		Name:           "cancel non-existent draft returns 404",
		Method:         http.MethodPost,
		URL:            "/api/drafts/nonexistent_id/cancel",
		ExpectedStatus: 404,
		ExpectedContent: []string{
			`"error"`,
			`"draft not found"`,
		},
		TestAppFactory: draftTestAppFactory,
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			cols := setupDraftTestCollections(t, app)
			_, token := createTestUser(t, app, cols.users, "user@test.com")
			s.Headers = map[string]string{"Authorization": token}
			RegisterDraftRoutes(e, nil, nil, nil)
		},
	}
	s.Test(t)
}

func TestCancelDraft_NotOwner(t *testing.T) {
	var s tests.ApiScenario
	s = tests.ApiScenario{
		Name:           "cancel draft owned by another user returns 403",
		Method:         http.MethodPost,
		ExpectedStatus: 403,
		ExpectedContent: []string{
			`"error"`,
			`"forbidden"`,
		},
		TestAppFactory: draftTestAppFactory,
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			cols := setupDraftTestCollections(t, app)
			owner, _ := createTestUser(t, app, cols.users, "owner@test.com")
			_, otherToken := createTestUser(t, app, cols.users, "other@test.com")
			draft := createTestDraft(t, app, cols.drafts, owner.Id, "queued")
			s.URL = "/api/drafts/" + draft.Id + "/cancel"
			s.Headers = map[string]string{"Authorization": otherToken}
			RegisterDraftRoutes(e, nil, nil, nil)
		},
	}
	s.Test(t)
}

func TestCancelDraft_NotQueued(t *testing.T) {
	var s tests.ApiScenario
	s = tests.ApiScenario{
		Name:           "cancel draft with status 'draft' returns 400",
		Method:         http.MethodPost,
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"error"`,
			`"only queued drafts can be cancelled"`,
		},
		TestAppFactory: draftTestAppFactory,
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			cols := setupDraftTestCollections(t, app)
			user, token := createTestUser(t, app, cols.users, "user@test.com")
			draft := createTestDraft(t, app, cols.drafts, user.Id, "draft")
			s.URL = "/api/drafts/" + draft.Id + "/cancel"
			s.Headers = map[string]string{"Authorization": token}
			RegisterDraftRoutes(e, nil, nil, nil)
		},
	}
	s.Test(t)
}

func TestCancelDraft_Success(t *testing.T) {
	var s tests.ApiScenario
	s = tests.ApiScenario{
		Name:           "cancel queued draft succeeds",
		Method:         http.MethodPost,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"status":"draft"`,
		},
		TestAppFactory: draftTestAppFactory,
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			cols := setupDraftTestCollections(t, app)
			user, token := createTestUser(t, app, cols.users, "user@test.com")
			draft := createTestDraft(t, app, cols.drafts, user.Id, "queued")
			s.URL = "/api/drafts/" + draft.Id + "/cancel"
			s.Headers = map[string]string{"Authorization": token}
			RegisterDraftRoutes(e, nil, nil, nil)
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			drafts, err := app.FindRecordsByFilter("drafts", "", "", 10, 0)
			if err != nil {
				t.Fatalf("failed to find drafts: %v", err)
			}
			if len(drafts) != 1 {
				t.Fatalf("expected 1 draft, got %d", len(drafts))
			}
			if drafts[0].GetString("status") != "draft" {
				t.Errorf("expected status 'draft' in DB, got %q", drafts[0].GetString("status"))
			}
		},
	}
	s.Test(t)
}

func TestEditDraft_NotFound(t *testing.T) {
	var s tests.ApiScenario
	s = tests.ApiScenario{
		Name:           "edit non-existent draft returns 404",
		Method:         http.MethodPatch,
		URL:            "/api/drafts/nonexistent_id",
		Body:           strings.NewReader(`{"edited_text":"updated"}`),
		ExpectedStatus: 404,
		ExpectedContent: []string{
			`"error"`,
			`"draft not found"`,
		},
		TestAppFactory: draftTestAppFactory,
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			cols := setupDraftTestCollections(t, app)
			_, token := createTestUser(t, app, cols.users, "user@test.com")
			s.Headers = map[string]string{"Authorization": token}
			RegisterDraftRoutes(e, nil, nil, nil)
		},
	}
	s.Test(t)
}

func TestEditDraft_NotOwner(t *testing.T) {
	var s tests.ApiScenario
	s = tests.ApiScenario{
		Name:           "edit draft owned by another user returns 403",
		Method:         http.MethodPatch,
		Body:           strings.NewReader(`{"edited_text":"updated"}`),
		ExpectedStatus: 403,
		ExpectedContent: []string{
			`"error"`,
			`"forbidden"`,
		},
		TestAppFactory: draftTestAppFactory,
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			cols := setupDraftTestCollections(t, app)
			owner, _ := createTestUser(t, app, cols.users, "owner@test.com")
			_, otherToken := createTestUser(t, app, cols.users, "other@test.com")
			draft := createTestDraft(t, app, cols.drafts, owner.Id, "draft")
			s.URL = "/api/drafts/" + draft.Id
			s.Headers = map[string]string{"Authorization": otherToken}
			RegisterDraftRoutes(e, nil, nil, nil)
		},
	}
	s.Test(t)
}

func TestEditDraft_Success(t *testing.T) {
	var s tests.ApiScenario
	s = tests.ApiScenario{
		Name:           "edit draft updates edited_text",
		Method:         http.MethodPatch,
		Body:           strings.NewReader(`{"edited_text":"my custom edit"}`),
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"edited_text":"my custom edit"`,
		},
		TestAppFactory: draftTestAppFactory,
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			cols := setupDraftTestCollections(t, app)
			user, token := createTestUser(t, app, cols.users, "user@test.com")
			draft := createTestDraft(t, app, cols.drafts, user.Id, "draft")
			s.URL = "/api/drafts/" + draft.Id
			s.Headers = map[string]string{"Authorization": token}
			RegisterDraftRoutes(e, nil, nil, nil)
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
			drafts, err := app.FindRecordsByFilter("drafts", "", "", 10, 0)
			if err != nil {
				t.Fatalf("failed to find drafts: %v", err)
			}
			if len(drafts) != 1 {
				t.Fatalf("expected 1 draft, got %d", len(drafts))
			}
			if drafts[0].GetString("edited_text") != "my custom edit" {
				t.Errorf("expected edited_text 'my custom edit' in DB, got %q", drafts[0].GetString("edited_text"))
			}
		},
	}
	s.Test(t)
}
