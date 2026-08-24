# Current Next Action

**Chunk G3** – Simplify the install template so only Title + Admin fields are visible.

### Already done in this session
- Environment capacity restored
- Cloned Forgejo v16.0.3
- Created branch `forgejo-simple-minimal`
- Applied first real code change: `getSupportedDbTypeNames()` now only returns SQLite3

Next small step: edit `templates/install.tmpl` to hide the database selection UI and most other fields.
