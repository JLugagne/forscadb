//go:build e2e

package e2eapi_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/JLugagne/forscadb/internal/dataforge/app"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/nosql"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/memory"
	"github.com/JLugagne/forscadb/internal/domain"
)

// newWiredStack creates a fresh ConnectionManager + NoSQLService backed by a
// real MongoDB testcontainer.  Each test gets its own in-memory connstore so
// tests are fully isolated.
func newWiredStack(t *testing.T) (*app.ConnectionManager, *app.NoSQLService) {
	t.Helper()
	store := memory.NewConnStore()
	factory := outbound.NewFactory()
	manager := app.NewConnectionManager(store, factory)
	svc := app.NewNoSQLService(manager)
	return manager, svc
}

// createAndConnect registers a new MongoDB connection in the manager and
// connects it, returning the assigned connection ID.
func createAndConnect(t *testing.T, ctx context.Context, manager *app.ConnectionManager, database string) string {
	t.Helper()
	conn, err := manager.Create(ctx, connection.Connection{
		Name:     "test-mongo",
		Engine:   domain.EngineMongoDB,
		Category: domain.CategoryNoSQL,
		Host:     mongoHost,
		Port:     mongoPort,
		Database: database,
	})
	if err != nil {
		t.Fatalf("Create connection: %v", err)
	}

	if err := manager.Connect(ctx, conn.ID); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	return conn.ID
}

// seedUsers inserts the canonical user documents and returns them (each has
// an _id added by InsertDocument).
func seedUsers(t *testing.T, ctx context.Context, svc *app.NoSQLService, connID string) []nosql.Document {
	t.Helper()
	users := []nosql.Document{
		{"email": "alice@test.com", "username": "alice", "role": "admin", "status": "active"},
		{"email": "bob@test.com", "username": "bob", "role": "user", "status": "active"},
		{"email": "carol@test.com", "username": "carol", "role": "user", "status": "inactive"},
	}
	inserted := make([]nosql.Document, 0, len(users))
	for _, u := range users {
		doc, err := svc.InsertDocument(ctx, connID, "users", u)
		if err != nil {
			t.Fatalf("seed users InsertDocument: %v", err)
		}
		inserted = append(inserted, doc)
	}
	return inserted
}

// seedProducts inserts the canonical product documents.
func seedProducts(t *testing.T, ctx context.Context, svc *app.NoSQLService, connID string) {
	t.Helper()
	products := []nosql.Document{
		{"name": "Widget", "sku": "WDG-001", "price": 29.99, "category": "tools"},
		{"name": "Gadget", "sku": "GDG-001", "price": 49.99, "category": "electronics"},
	}
	for _, p := range products {
		if _, err := svc.InsertDocument(ctx, connID, "products", p); err != nil {
			t.Fatalf("seed products InsertDocument: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMongoConnection(t *testing.T) {
	ctx := context.Background()
	manager, _ := newWiredStack(t)

	conn, err := manager.Create(ctx, connection.Connection{
		Name:     "test-conn",
		Engine:   domain.EngineMongoDB,
		Category: domain.CategoryNoSQL,
		Host:     mongoHost,
		Port:     mongoPort,
		Database: "conntest",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// After creation the connection must be disconnected.
	got, err := manager.Get(ctx, conn.ID)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.Status != domain.StatusDisconnected {
		t.Errorf("expected status %q after Create, got %q", domain.StatusDisconnected, got.Status)
	}

	// Connect.
	if err := manager.Connect(ctx, conn.ID); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	got, err = manager.Get(ctx, conn.ID)
	if err != nil {
		t.Fatalf("Get after Connect: %v", err)
	}
	if got.Status != domain.StatusConnected {
		t.Errorf("expected status %q after Connect, got %q", domain.StatusConnected, got.Status)
	}

	// Disconnect.
	if err := manager.Disconnect(ctx, conn.ID); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	got, err = manager.Get(ctx, conn.ID)
	if err != nil {
		t.Fatalf("Get after Disconnect: %v", err)
	}
	if got.Status != domain.StatusDisconnected {
		t.Errorf("expected status %q after Disconnect, got %q", domain.StatusDisconnected, got.Status)
	}
}

func TestMongoGetCollections(t *testing.T) {
	ctx := context.Background()
	manager, svc := newWiredStack(t)
	connID := createAndConnect(t, ctx, manager, "colltest")

	seedUsers(t, ctx, svc, connID)
	seedProducts(t, ctx, svc, connID)

	collections, err := svc.GetCollections(ctx, connID)
	if err != nil {
		t.Fatalf("GetCollections: %v", err)
	}

	byName := make(map[string]nosql.Collection)
	for _, c := range collections {
		byName[c.Name] = c
	}

	for _, want := range []string{"users", "products"} {
		c, ok := byName[want]
		if !ok {
			t.Errorf("collection %q not found in GetCollections result", want)
			continue
		}

		// Verify document counts.
		switch want {
		case "users":
			if c.DocumentCount != 3 {
				t.Errorf("users: expected DocumentCount 3, got %d", c.DocumentCount)
			}
		case "products":
			if c.DocumentCount != 2 {
				t.Errorf("products: expected DocumentCount 2, got %d", c.DocumentCount)
			}
		}

		// Every collection must have at least the _id_ index.
		hasIDIndex := false
		for _, idx := range c.Indexes {
			if idx.Name == "_id_" {
				hasIDIndex = true
				break
			}
		}
		if !hasIDIndex {
			t.Errorf("collection %q: _id_ index not found in %v", want, c.Indexes)
		}
	}
}

func TestMongoGetDocuments(t *testing.T) {
	ctx := context.Background()
	manager, svc := newWiredStack(t)
	connID := createAndConnect(t, ctx, manager, "getdocstest")

	seedUsers(t, ctx, svc, connID)

	t.Run("empty filter returns all documents", func(t *testing.T) {
		docs, err := svc.GetDocuments(ctx, connID, "users", "", 0)
		if err != nil {
			t.Fatalf("GetDocuments (empty filter): %v", err)
		}
		if len(docs) != 3 {
			t.Errorf("expected 3 documents, got %d", len(docs))
		}
	})

	t.Run("filter by role admin returns only alice", func(t *testing.T) {
		docs, err := svc.GetDocuments(ctx, connID, "users", `{"role":"admin"}`, 0)
		if err != nil {
			t.Fatalf("GetDocuments (role=admin filter): %v", err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}
		username, ok := docs[0]["username"].(string)
		if !ok || username != "alice" {
			t.Errorf("expected username %q, got %v", "alice", docs[0]["username"])
		}
	})

	t.Run("limit 1 returns exactly 1 document", func(t *testing.T) {
		docs, err := svc.GetDocuments(ctx, connID, "users", "", 1)
		if err != nil {
			t.Fatalf("GetDocuments (limit=1): %v", err)
		}
		if len(docs) != 1 {
			t.Errorf("expected 1 document with limit 1, got %d", len(docs))
		}
	})

	t.Run("_id field is a string", func(t *testing.T) {
		docs, err := svc.GetDocuments(ctx, connID, "users", "", 1)
		if err != nil {
			t.Fatalf("GetDocuments (_id check): %v", err)
		}
		if len(docs) == 0 {
			t.Fatal("no documents returned")
		}
		id, ok := docs[0]["_id"].(string)
		if !ok {
			t.Errorf("expected _id to be a string, got %T (%v)", docs[0]["_id"], docs[0]["_id"])
		} else if id == "" {
			t.Error("expected non-empty _id string")
		}
	})
}

func TestMongoInsertDocument(t *testing.T) {
	ctx := context.Background()
	manager, svc := newWiredStack(t)
	connID := createAndConnect(t, ctx, manager, "inserttest")

	newDoc := nosql.Document{
		"email":    "dave@test.com",
		"username": "dave",
		"role":     "user",
		"status":   "active",
	}

	inserted, err := svc.InsertDocument(ctx, connID, "users", newDoc)
	if err != nil {
		t.Fatalf("InsertDocument: %v", err)
	}

	// Returned document must contain a non-empty _id string.
	id, ok := inserted["_id"].(string)
	if !ok {
		t.Fatalf("expected _id to be a string, got %T", inserted["_id"])
	}
	if id == "" {
		t.Fatal("inserted document has empty _id")
	}

	// Document must appear when queried back.
	docs, err := svc.GetDocuments(ctx, connID, "users", `{"username":"dave"}`, 0)
	if err != nil {
		t.Fatalf("GetDocuments after InsertDocument: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document after insert, got %d", len(docs))
	}
	if docs[0]["email"] != "dave@test.com" {
		t.Errorf("expected email %q, got %v", "dave@test.com", docs[0]["email"])
	}
}

func TestMongoUpdateDocument(t *testing.T) {
	ctx := context.Background()
	manager, svc := newWiredStack(t)
	connID := createAndConnect(t, ctx, manager, "updatetest")

	// Insert a document and grab its _id.
	inserted, err := svc.InsertDocument(ctx, connID, "users", nosql.Document{
		"email":    "eve@test.com",
		"username": "eve",
		"role":     "user",
		"status":   "active",
	})
	if err != nil {
		t.Fatalf("InsertDocument (setup): %v", err)
	}
	id, ok := inserted["_id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected string _id from InsertDocument, got %T %v", inserted["_id"], inserted["_id"])
	}

	// Update the document.
	updated, err := svc.UpdateDocument(ctx, connID, "users", id, nosql.Document{
		"email":    "eve@test.com",
		"username": "eve",
		"role":     "admin",
		"status":   "active",
	})
	if err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}
	_ = updated

	// Verify the change is visible via GetDocuments.
	filter := fmt.Sprintf(`{"_id":{"$oid":"%s"}}`, id)
	docs, err := svc.GetDocuments(ctx, connID, "users", filter, 0)
	if err != nil {
		// Fallback: retrieve all and look for the id.
		docs, err = svc.GetDocuments(ctx, connID, "users", "", 0)
		if err != nil {
			t.Fatalf("GetDocuments after UpdateDocument: %v", err)
		}
	}

	var found nosql.Document
	for _, d := range docs {
		if docID, _ := d["_id"].(string); docID == id {
			found = d
			break
		}
	}
	if found == nil {
		t.Fatalf("could not find updated document with _id %q", id)
	}
	if found["role"] != "admin" {
		t.Errorf("expected role %q after update, got %v", "admin", found["role"])
	}
}

func TestMongoCreateCollection(t *testing.T) {
	ctx := context.Background()
	manager, svc := newWiredStack(t)
	connID := createAndConnect(t, ctx, manager, "createcolltest")

	if err := svc.CreateCollection(ctx, connID, "test_create_coll"); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	collections, err := svc.GetCollections(ctx, connID)
	if err != nil {
		t.Fatalf("GetCollections after CreateCollection: %v", err)
	}

	found := false
	for _, c := range collections {
		if c.Name == "test_create_coll" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, 0, len(collections))
		for _, c := range collections {
			names = append(names, c.Name)
		}
		t.Errorf("test_create_coll not found after CreateCollection; got: %v", names)
	}
}

func TestMongoDropCollection(t *testing.T) {
	ctx := context.Background()
	manager, svc := newWiredStack(t)
	connID := createAndConnect(t, ctx, manager, "dropcolltest")

	// Insert a document to implicitly create the collection.
	if _, err := svc.InsertDocument(ctx, connID, "test_drop_coll", nosql.Document{"seed": "value"}); err != nil {
		t.Fatalf("InsertDocument (setup): %v", err)
	}

	// Verify the collection exists.
	collections, err := svc.GetCollections(ctx, connID)
	if err != nil {
		t.Fatalf("GetCollections before drop: %v", err)
	}
	exists := false
	for _, c := range collections {
		if c.Name == "test_drop_coll" {
			exists = true
			break
		}
	}
	if !exists {
		t.Fatal("test_drop_coll should exist before DropCollection")
	}

	// Drop the collection.
	if err := svc.DropCollection(ctx, connID, "test_drop_coll"); err != nil {
		t.Fatalf("DropCollection: %v", err)
	}

	// Verify the collection is gone.
	collections, err = svc.GetCollections(ctx, connID)
	if err != nil {
		t.Fatalf("GetCollections after drop: %v", err)
	}
	for _, c := range collections {
		if c.Name == "test_drop_coll" {
			t.Error("test_drop_coll still present after DropCollection")
		}
	}
}

func TestMongoDeleteDocument(t *testing.T) {
	ctx := context.Background()
	manager, svc := newWiredStack(t)
	connID := createAndConnect(t, ctx, manager, "deletetest")

	// Insert a document.
	inserted, err := svc.InsertDocument(ctx, connID, "users", nosql.Document{
		"email":    "frank@test.com",
		"username": "frank",
		"role":     "user",
		"status":   "active",
	})
	if err != nil {
		t.Fatalf("InsertDocument (setup): %v", err)
	}
	id, ok := inserted["_id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected string _id from InsertDocument, got %T %v", inserted["_id"], inserted["_id"])
	}

	// Delete the document.
	if err := svc.DeleteDocument(ctx, connID, "users", id); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}

	// The document must no longer appear in GetDocuments.
	docs, err := svc.GetDocuments(ctx, connID, "users", "", 0)
	if err != nil {
		t.Fatalf("GetDocuments after DeleteDocument: %v", err)
	}
	for _, d := range docs {
		if docID, _ := d["_id"].(string); docID == id {
			t.Errorf("document with _id %q still present after DeleteDocument", id)
		}
	}
}
