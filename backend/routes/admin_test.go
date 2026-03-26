package routes

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func setupUsersCollection(t *testing.T, app *tests.TestApp) *core.Collection {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("users")
	if err == nil {
		existing, _ := app.FindRecordsByFilter("users", "", "", 0, 0)
		for _, r := range existing {
			_ = app.Delete(r)
		}
		return col
	}
	col = core.NewAuthCollection("users")
	if err := app.Save(col); err != nil {
		t.Fatalf("failed to create users collection: %v", err)
	}
	return col
}

func createUser(t *testing.T, app *tests.TestApp, col *core.Collection, email, password string) *core.Record {
	t.Helper()
	record := core.NewRecord(col)
	record.SetEmail(email)
	record.SetPassword(password)
	record.Set("verified", true)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to create user %s: %v", email, err)
	}
	return record
}

func TestIsSuperuser_NilAuthRecord(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	setupUsersCollection(t, app)

	if isSuperuser(app, nil) {
		t.Error("expected isSuperuser(nil) to return false")
	}
}

func TestIsSuperuser_FirstCreatedUser(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	col := setupUsersCollection(t, app)
	first := createUser(t, app, col, "admin@test.com", "password123456")

	// Create a second user after a brief pause to ensure different created timestamps
	time.Sleep(10 * time.Millisecond)
	createUser(t, app, col, "other@test.com", "password123456")

	if !isSuperuser(app, first) {
		t.Error("expected first created user to be superuser")
	}
}

func TestIsSuperuser_NotFirstCreatedUser(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	col := setupUsersCollection(t, app)
	createUser(t, app, col, "admin@test.com", "password123456")

	time.Sleep(10 * time.Millisecond)
	second := createUser(t, app, col, "other@test.com", "password123456")

	if isSuperuser(app, second) {
		t.Error("expected non-first user to NOT be superuser")
	}
}

func TestIsSuperuser_SameEmailDifferentRecord(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	col := setupUsersCollection(t, app)
	first := createUser(t, app, col, "admin@test.com", "password123456")

	// Create a fake record with the same email but different ID to test the || email check.
	// We simulate this by creating a new in-memory record with the first user's email
	// but a different ID.
	fakeRecord := core.NewRecord(col)
	fakeRecord.SetEmail(first.Email())
	fakeRecord.Id = "fake_different_id"

	if !isSuperuser(app, fakeRecord) {
		t.Error("expected record with same email as first user to be superuser (email fallback)")
	}
}

func TestIsSuperuser_NoUsersInCollection(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	col := setupUsersCollection(t, app)

	// Create a record in memory (not saved) to pass as authRecord
	record := core.NewRecord(col)
	record.SetEmail("ghost@test.com")
	record.Id = "ghost_id"

	if isSuperuser(app, record) {
		t.Error("expected isSuperuser to return false when no users exist in collection")
	}
}
