# Current Next Action

**Chunk G3** – Simplify the install template (`templates/install.tmpl`).

### Status of this attempt
Environment capacity disappeared again while starting G3.

### Already achieved in the previous successful window
- Cloned Forgejo v16.0.3
- Branch `forgejo-simple-minimal` created
- Real code change committed: SQLite3 forced as the only database type in `routers/install/install.go`

### What G3 will do when capacity returns
Edit `templates/install.tmpl` so that:
- Database type selector is removed / hidden
- Most server and path fields are hidden
- Only Instance Title + Admin Username / Password / Confirm remain visible
- Hidden inputs supply the SQLite defaults

Say **Next** to retry the environment and continue G3.
