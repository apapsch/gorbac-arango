# RBAC ArangoDB Backend

The venerable library
[github.com/mikespook/gorbac](https://github.com/mikespook/gorbac)
provides a polished small interface for RBAC (role based access
control), though leaves persistence of the RBAC structure as an
exercise for the user. This library provides persistence for ArangoDB.

## Usage

Create required collections with `rbacarango.CreateSchema` once in your schema
migration tool. The method is replayable.

```
func CreateSchema(ctx context.Context, arangoDB driver.Database) error
```

Then `rbacarango.LoadRBAC` or `rbacarango.SaveRBAC` with the RBAC
structure from `gorbac` package.

```
func LoadRBAC(ctx context.Context, db driver.Database) (*gorbac.RBAC, error)
func SaveRBAC(ctx context.Context, db driver.Database, rbac *gorbac.RBAC) error
```

## License

SPDX short identifier: MIT
