# Ready-to-apply changes for routers/install/install.go

## 1. Force SQLite-only (already applied once in a previous session)

Replace the function `getSupportedDbTypeNames` with:

```go
func getSupportedDbTypeNames() (dbTypeNames []map[string]string) {
	// forgejo-simple: only offer SQLite3
	dbTypeNames = append(dbTypeNames, map[string]string{"type": "sqlite3", "name": setting.DatabaseTypeNames["sqlite3"]})
	return dbTypeNames
}
```

## 2. In the InstallPost handler

At the beginning of form processing, force:

```go
form.DbType = "sqlite3"
if form.DbPath == "" {
	form.DbPath = filepath.Join(setting.AppDataPath, "forgejo.db")
}
```

Skip any MySQL / PostgreSQL connection test blocks.

Only require and validate:
- `form.AdminName`
- `form.AdminPasswd` + confirmation match

Everything else receives the hidden defaults from the simplified template.
