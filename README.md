# forgejo-simple

**Faster. Simpler. No bloat.**

A community fork / hard fork direction of [Forgejo](https://forgejo.org) aimed at people who want a GitHub alternative that is:

- Extremely lightweight
- SQLite-first (and SQLite-only by default)
- One-click (or zero-config) install
- Free of unnecessary features and UI steps

## Why

Forgejo is already excellent. This project exists to push it further toward radical simplicity:

- No MySQL / PostgreSQL required for 95% of users
- Install wizard reduced from ~15 fields to 3–4
- Opinionated defaults so you can be productive in seconds
- Easy path to strip packages, actions, projects, etc. later

## Current Status (2026-08-24)

- Source of Forgejo v16.0.3 was previously cloned and a live instance was started in a test environment.
- Live session has been cleared.
- First concrete improvement is defined below and will be applied to a real fork as soon as a full build environment is available again.

## Planned First Change: SQLite-only + Minimal Install Wizard

### Goals
1. Hard-code SQLite3 as the only database option.
2. Hide or remove every non-essential field from the install form.
3. Pre-fill sensible defaults for repository path, LFS, domain, etc.
4. Require only: Instance title, Admin username, Admin password.

### Files that will be modified (from upstream Forgejo)

- `templates/install.tmpl` (or the equivalent install template)
- `routers/install/install.go` (form handling + validation)
- `modules/setting/database.go` / database related settings
- Default `app.ini` generation logic

### Result after the change
- New installs become dramatically faster and less intimidating.
- Aligns with the original vision of a GitHub alternative without unneeded steps.

## Next Steps

1. Re-acquire a working build environment.
2. Apply the SQLite-only + simplified wizard patches.
3. Add a `MINIMAL=true` mode that further disables optional features.
4. Eventually provide a true single-binary zero-config mode.

## License

Same as Forgejo (GPL-3.0-or-later) once the actual fork code lands.

---

Maintained by [@5mil](https://github.com/5mil)  
Inspired by the desire for a free, open, fast, and simple Git hosting platform.
