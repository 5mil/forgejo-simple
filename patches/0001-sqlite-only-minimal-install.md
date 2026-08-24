# Patch 0001 – SQLite-only + Minimal Install Wizard

Target: Forgejo v16.x

## Goal
Reduce the install page from ~15 fields to 4 visible fields and force SQLite.

---

## 1. Install Template (before → after)

### Before (simplified view of current upstream)
- Database type dropdown (MySQL / PostgreSQL / SQLite / …)
- Host, Username, Password, Database name
- Instance title, slogan
- Repository root path, LFS path
- User to run as
- Domain, SSH port, HTTP port, App URL
- Log paths, etc.
- Admin account fields

### After (what we want)

Only these visible fields:

```html
<form method="post">
  {{.CsrfTokenHtml}}

  <fieldset>
    <legend>Instance</legend>
    <label>Title</label>
    <input name="app_name" value="Forgejo Simple" required>
  </fieldset>

  <fieldset>
    <legend>Administrator</legend>
    <label>Username</label>
    <input name="admin_name" required autocomplete="username">

    <label>Password</label>
    <input type="password" name="admin_passwd" required autocomplete="new-password">

    <label>Confirm Password</label>
    <input type="password" name="admin_confirm_passwd" required autocomplete="new-password">

    <label>Email (optional)</label>
    <input type="email" name="admin_email" autocomplete="email">
  </fieldset>

  <!-- All other values become hidden defaults -->
  <input type="hidden" name="db_type" value="sqlite3">
  <input type="hidden" name="db_path" value="data/forgejo.db">
  <input type="hidden" name="repo_root_path" value="data/forgejo-repositories">
  <input type="hidden" name="lfs_root_path" value="data/lfs">
  <input type="hidden" name="run_user" value="git">
  <input type="hidden" name="domain" value="localhost">
  <input type="hidden" name="ssh_port" value="22">
  <input type="hidden" name="http_port" value="3000">
  <input type="hidden" name="app_url" value="http://localhost:3000/">

  <button type="submit">Install</button>
</form>
```

---

## 2. Install Handler (Go) – key changes

### Before
- Reads `db_type` from form
- Runs different connection tests for MySQL / Postgres / SQLite
- Validates many optional fields

### After (logic we want)

```go
// Force SQLite regardless of anything else
form.DbType = "sqlite3"
form.DbPath = filepath.Join(setting.AppDataPath, "forgejo.db")

// Only validate the fields we still show
if form.AdminName == "" {
    ctx.RenderWithErr("Admin username is required", tplInstall, &form)
    return
}
if form.AdminPasswd == "" || form.AdminPasswd != form.AdminConfirmPasswd {
    ctx.RenderWithErr("Password is required and must match", tplInstall, &form)
    return
}

// Skip all MySQL/Postgres connection logic
// Generate minimal app.ini and create the admin user
```

---

## 3. Generated app.ini (result)

```ini
APP_NAME = Forgejo Simple
RUN_MODE = prod

[server]
DOMAIN    = localhost
HTTP_PORT = 3000
ROOT_URL  = http://localhost:3000/

[database]
DB_TYPE = sqlite3
PATH    = data/forgejo.db

[repository]
ROOT = data/forgejo-repositories

[lfs]
PATH = data/lfs

[security]
INSTALL_LOCK = true
```

---

## Success criteria for this patch
- Install page shows only 4 fields
- No database type selection appears
- After clicking Install the instance is immediately usable with SQLite
