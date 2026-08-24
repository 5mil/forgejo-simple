# forgejo-simple

**A truly minimal Git forge.**

Faster. Simpler. No bloat.

This project started as an attempt to simplify Forgejo.  
We have now pivoted to building a **small forge from scratch** that only includes what is actually needed:

- Host Git repositories
- Extremely simple web UI (list repos, browse files, view commits)
- SQLite (or pure filesystem) for metadata
- Single static binary
- Almost zero configuration

No packages, no Actions, no Projects, no heavy frontend, no long install wizard.

---

## Status

We are at the very beginning of the from-scratch implementation.

| Piece                    | Status   |
|--------------------------|----------|
| Project direction        | Done     |
| Basic Go module          | Next     |
| Git smart HTTP endpoint  | Planned |
| Simple web UI            | Planned |
| SQLite metadata          | Planned |
| Single binary release    | Planned |

---

## Design principles

1. **One binary** – `./forgejo-simple` should just work.
2. **SQLite or filesystem first** – no external database required.
3. **Server-rendered HTML** – minimal or zero JavaScript.
4. **Git is the source of truth** – we do not re-implement version control.
5. **Sensible defaults** – almost no configuration needed for local use.

---

## Repository layout (growing)

```
forgejo-simple/
├── README.md
├── go.mod                  ← coming next
├── cmd/                   ← main entrypoint
├── internal/
│   ├── git/                ← smart HTTP + repo helpers
│   ├── web/                ← simple HTML UI
│   └── db/                 ← optional SQLite metadata
└── data/                  ← repositories live here by default
```

---

## License

MIT (for the new minimal code).  
Any future optional Forgejo-derived patches will remain under GPL-3.0-or-later.

---

Maintained by [@5mil](https://github.com/5mil)
