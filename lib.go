// Copyright 2021 Stern Data GmbH

// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use, copy,
// modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
// BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
// ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package rbacarango

import (
	"context"
	_ "embed"
	"fmt"

	driver "github.com/arangodb/go-driver"
	"gopkg.in/mikespook/gorbac.v2"
)

const (
	graphName                  = "gorbac"
	roleCollectionName         = "gorbac_roles"
	permissionCollectionName   = "gorbac_permissions"
	rolePermEdgeCollectionName = "gorbac_roles_permissions"
)

//go:embed queries/load.aql
var loadQueryContents string

// CreateSchema creates graph collections for edges and vertices.
func CreateSchema(ctx context.Context, db driver.Database) error {
	exists, err := db.CollectionExists(ctx, roleCollectionName)
	if err != nil {
		return err
	}
	if !exists {
		_, err = db.CreateCollection(ctx, roleCollectionName, nil)
		if err != nil {
			return err
		}
	}

	exists, err = db.CollectionExists(ctx, permissionCollectionName)
	if err != nil {
		return err
	}
	if !exists {
		_, err = db.CreateCollection(ctx, permissionCollectionName, nil)
		if err != nil {
			return err
		}
	}

	exists, err = db.GraphExists(ctx, graphName)
	if err != nil {
		return err
	}
	if !exists {
		_, err = db.CreateGraph(ctx, graphName, &driver.CreateGraphOptions{
			EdgeDefinitions: []driver.EdgeDefinition{
				{
					Collection: rolePermEdgeCollectionName,
					From:       []string{roleCollectionName},
					To:         []string{permissionCollectionName},
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type rbacCollections struct {
	Roles       driver.Collection
	Permissions driver.Collection
	Edges       driver.Collection
}

func newRBACCollections(ctx context.Context, db driver.Database) (*rbacCollections, error) {
	graph, err := db.Graph(ctx, graphName)
	if err != nil {
		return nil, err
	}

	roleCollection, err := graph.VertexCollection(ctx, roleCollectionName)
	if err != nil {
		return nil, err
	}

	permCollection, err := graph.VertexCollection(ctx, permissionCollectionName)
	if err != nil {
		return nil, err
	}

	edges, _, err := graph.EdgeCollection(ctx, rolePermEdgeCollectionName)
	if err != nil {
		return nil, err
	}

	return &rbacCollections{
		Roles:       roleCollection,
		Permissions: permCollection,
		Edges:       edges,
	}, nil
}

type keyed struct {
	Key string `json:"_key"`
}

func createKeyed(ctx context.Context, col driver.Collection, key string) (err error) {
	var exists bool
	exists, err = col.DocumentExists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		_, err = col.CreateDocument(ctx, map[string]string{
			"_key": key,
		})
	}
	return
}

// SaveRBAC stores whole RBAC structure.
func SaveRBAC(ctx context.Context, db driver.Database, rbac *gorbac.RBAC) error {
	collections, err := newRBACCollections(ctx, db)
	if err != nil {
		return err
	}
	return gorbac.Walk(rbac, func(r gorbac.Role, parents []string) error {
		err := createKeyed(ctx, collections.Roles, r.ID())
		if err != nil {
			return err
		}

		stdRole, ok := r.(*gorbac.StdRole)
		if !ok {
			return fmt.Errorf("Role is not a gorbac.StdRole")
		}

		for _, p := range stdRole.Permissions() {
			err = createKeyed(ctx, collections.Permissions, p.ID())
			if err != nil {
				return err
			}
			_, err = collections.Edges.CreateDocument(ctx, map[string]string{
				"_from": roleCollectionName + "/" + r.ID(),
				"_to":   permissionCollectionName + "/" + p.ID(),
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

type loadedRole struct {
	RoleID      string   `json:"r"`
	Permissions []string `json:"p"`
}

// LoadRBAC loads whole RBAC structure.
func LoadRBAC(ctx context.Context, db driver.Database) (*gorbac.RBAC, error) {
	cursor, err := db.Query(ctx, loadQueryContents, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()

	rbac := gorbac.New()
	for {
		if !cursor.HasMore() {
			break
		}

		var l loadedRole
		_, err = cursor.ReadDocument(ctx, &l)
		if err != nil {
			return nil, err
		}

		role := gorbac.NewStdRole(l.RoleID)
		for _, p := range l.Permissions {
			role.Assign(gorbac.NewStdPermission(p))
		}

		rbac.Add(role)
	}
	return rbac, nil
}
