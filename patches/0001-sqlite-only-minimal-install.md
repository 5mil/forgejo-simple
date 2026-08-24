# Patch 0001 – SQLite-only + Minimal Install Wizard

This document describes the exact changes needed in upstream Forgejo (v16.x) to implement the first simplification.

## 1. templates/install.tmpl (or equivalent)

Replace the large database + server settings form with a minimal version:

```html
<form method="post" action="{{AppSubUrl}}/install">
  {{.CsrfTokenHtml}}

  <h2>Instance</h2>
  <label>Title</label>
  <input name="app_name" value="Forgejo Simple" required>

  <h2>Administrator Account</h2>
  <label>Username</label>
  <input name="admin_name" required>

  <label>Password</label>
  <input type="password" name="admin_passwd" required>

  <label>Confirm Password</label>
  <input type="password" name="admin_confirm_passwd" required>

  <label>Email (optional)</label>
  <input type="email" name="admin_email">

  <!-- Hidden defaults -->
  <input type="hidden" name="db_type" value="sqlite3">
  <input type="hidden" name="db_path" value="data/forgejo.db">
  <input type="hidden" name="repo_root_path" value="data/forgejo-repositories">
  <input type="hidden" name="lfs_root_path" value="data/lfs">
  <input type="hidden" name="run_user" value="git">
  <input type="hidden" name="domain" value="localhost">
  <input type="hidden" name="ssh_port" value="22">
  <input type="hidden" name="http_port" value="3000">
  <input type="hidden" name="app_url" value="http://localhost:3000/">

  <button type="submit">Install Forgejo Simple</button>
</form>
```

## 2. routers/install/install.go

- Force `db_type = "sqlite3"` regardless of form input.
- Skip all MySQL/Postgres connection tests.
- Only validate admin username + password + password confirmation.
- Write a minimal app.ini with SQLite settings only.

Key logic change (pseudo):

```go
func InstallPost(ctx *context.Context) {
    // Force SQLite
    form.DbType = "sqlite3"
    form.DbPath = filepath.Join(setting.AppDataPath, "forgejo.db")

    // Only require these
    if form.AdminName == "" || form.AdminPasswd == "" {
        // error
    }
    if form.AdminPasswd != form.AdminConfirmPasswd {
        // error
    }

    // Generate minimal config and create admin user
    // ...
}
```

## 3. Default app.ini generation

Always produce:

```ini
[database]
DB_TYPE = sqlite3
PATH = data/forgejo.db

[server]
DOMAIN = localhost
HTTP_PORT = 3000
ROOT_URL = http://localhost:3000/

[repository]
ROOT = data/forgejo-repositories

[lfs]
PATH = data/lfs
```

## Result

Install page goes from ~15 fields to 4 visible fields.  
One click → working forge.
