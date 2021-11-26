package rbacarango

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/arangodb/go-driver"
	arangohttp "github.com/arangodb/go-driver/http"
	"github.com/oklog/ulid"
	"gopkg.in/mikespook/gorbac.v2"
)

func databaseForTest(ctx context.Context) (driver.Database, error) {
	con, err := arangohttp.NewConnection(arangohttp.ConnectionConfig{
		Endpoints: []string{"http://127.0.0.1:8529"},
	})
	if err != nil {
		return nil, err
	}

	client, err := driver.NewClient(driver.ClientConfig{
		Connection: con,
	})
	if err != nil {
		return nil, err
	}

	dbID := ulid.MustNew(ulid.Now(), rand.Reader)
	dbName := "gotest_" + dbID.String()
	return client.CreateDatabase(ctx, dbName, nil)
}

func TestSaveLoadRBAC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := databaseForTest(ctx)
	if err != nil {
		t.Fatalf("arango test connection failed: %v", err)
	}

	err = CreateSchema(ctx, db)
	if err != nil {
		t.Fatalf("could not create schema: %v", err)
	}

	fooRole := gorbac.NewStdRole("foo")
	fooRole.Assign(gorbac.NewStdPermission("bar"))
	fooRole.Assign(gorbac.NewStdPermission("quux"))

	gandalfRole := gorbac.NewStdRole("gandalf")
	gandalfRole.Assign(gorbac.NewStdPermission("pass"))
	gandalfRole.Assign(gorbac.NewStdPermission("quux"))

	rbac := gorbac.New()
	rbac.Add(fooRole)
	rbac.Add(gandalfRole)

	err = SaveRBAC(ctx, db, rbac)
	if err != nil {
		t.Fatalf("could not save rbac: %v", err)
	}

	loaded, err := LoadRBAC(ctx, db)
	if err != nil {
		t.Fatalf("could not load rbac: %v", err)
	}

	if !gorbac.AnyGranted(loaded, []string{"foo"}, gorbac.NewStdPermission("bar"), nil) {
		t.Fatalf("foo role doesn't have bar permission")
	}

	if gorbac.AnyGranted(loaded, []string{"foo"}, gorbac.NewStdPermission("wtf"), nil) {
		t.Fatalf("foo role has wtf permission, shouldn't have")
	}

	if !gorbac.AllGranted(loaded, []string{"foo", "gandalf"}, gorbac.NewStdPermission("quux"), nil) {
		t.Fatalf("both foo and gandalf role don't have quux permission")
	}

	if gorbac.AllGranted(loaded, []string{"foo", "gandalf"}, gorbac.NewStdPermission("pass"), nil) {
		t.Fatalf("only gandalf shall have pass permission, but foo has too")
	}
}

const expectedCount = 1

func TestSaveTwiceAddedOnce(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := databaseForTest(ctx)
	if err != nil {
		t.Fatalf("arango test connection failed: %v", err)
	}

	err = CreateSchema(ctx, db)
	if err != nil {
		t.Fatalf("could not create schema: %v", err)
	}

	fooRole := gorbac.NewStdRole("foo")
	fooRole.Assign(gorbac.NewStdPermission("bar"))

	rbac := gorbac.New()
	rbac.Add(fooRole)

	err = SaveRBAC(ctx, db, rbac)
	if err != nil {
		t.Fatalf("could not save rbac: %v", err)
	}

	err = SaveRBAC(ctx, db, rbac)
	if err != nil {
		t.Fatalf("could not save rbac a second time: %v", err)
	}

	countedCollections := []string{
		roleCollectionName,
		permissionCollectionName,
		rolePermEdgeCollectionName,
	}
	for _, collectionName := range countedCollections {
		collection, err := db.Collection(ctx, collectionName)
		if err != nil {
			t.Fatalf("could not get collection %s: %v", collectionName, err)
		}
		count, err := collection.Count(ctx)
		if err != nil {
			t.Fatalf("could not count %s: %v", collectionName, err)
		}
		if count != expectedCount {
			t.Fatalf("expected %s count %d, got %d", collectionName, expectedCount, count)
		}
	}
}

func TestLoadRoleWithoutPermissions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := databaseForTest(ctx)
	if err != nil {
		t.Fatalf("arango test connection failed: %v", err)
	}

	err = CreateSchema(ctx, db)
	if err != nil {
		t.Fatalf("could not create schema: %v", err)
	}

	fooRole := gorbac.NewStdRole("quux")

	rbac := gorbac.New()
	rbac.Add(fooRole)

	err = SaveRBAC(ctx, db, rbac)
	if err != nil {
		t.Fatalf("could not save rbac: %v", err)
	}

	loaded, err := LoadRBAC(ctx, db)
	if err != nil {
		t.Fatalf("could not load rbac: %v", err)
	}

	found := false
	err = gorbac.Walk(loaded, func(r gorbac.Role, p []string) error {
		if r.ID() == fooRole.ID() {
			found = true
		}
		return nil
	})
	if !found {
		t.Fatalf("Walking RBAC structure did not include %s role", fooRole.ID())
	}
}
