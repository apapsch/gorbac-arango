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
