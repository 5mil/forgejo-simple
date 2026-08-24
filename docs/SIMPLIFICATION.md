# Simplification Plan

## Phase 1 – SQLite Only + Minimal Wizard (current focus)

### Changes

1. **Database**
   - Remove support for MySQL, MariaDB, PostgreSQL, TiDB from the install form.
   - Always use SQLite3.
   - Default path: `%(WORK_PATH)s/data/forgejo.db`

2. **Install Form Fields to Keep**
   - Instance Title (pre-filled with "Forgejo Simple")
   - Admin Username
   - Admin Password + Confirm
   - (Optional) Admin Email

3. **Fields to Hide / Hard-code**
   - Database type, host, user, password, name
   - Repository root path
   - LFS path
   - SSH domain / port
   - HTTP port / domain / root URL (use defaults + reverse-proxy friendly)
   - Log path, etc.

4. **Backend**
   - Skip all non-SQLite validation in the install handler.
   - Generate a clean minimal `app.ini` automatically.

### Expected UX

User opens the web UI → sees a short form → clicks Install → immediately has a working forge.

## Phase 2 (later)

- `MINIMAL=true` environment / config flag that disables:
  - Package registry
  - Actions / CI
  - Projects / Kanban
  - Wiki (optional)
- True zero-config mode: `./forgejo-simple` starts with no wizard at all for local use.
