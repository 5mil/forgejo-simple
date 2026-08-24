# New Direction – Minimal Forge from Scratch

We abandoned the approach of heavily patching Forgejo because:

1. The live build environment was unreliable.
2. Stripping a large existing codebase is slower than writing a focused small one.
3. The original goal was "faster, simpler, without bloat" – a clean small project matches that better.

## What we will build

**Phase 1 – Absolute minimum**
- Serve existing Git repositories over HTTP (smart HTTP protocol)
- Simple page that lists repositories
- Simple file browser and raw file serving
- Configuration via a single optional `config.ini` or just environment / flags

**Phase 2**
- Basic commit history view
- Simple diff view
- Create new repository from the UI (optional)

**Phase 3**
- Optional SQLite for stars, issues, or very lightweight metadata
- SSH access
- Authentication (optional, can stay public-first)

## Explicit non-goals (for a long time)
- CI / Actions
- Package registry
- Complex project boards
- Heavy JavaScript frontend
- Federation (can be revisited much later)
