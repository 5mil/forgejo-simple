# forgejo-simple

**Faster. Simpler. No bloat.**

A focused direction for Forgejo that prioritises:

- SQLite-only by default
- Extremely short install wizard
- Opinionated, minimal configuration
- No unnecessary features or steps

Repository: https://github.com/5mil/forgejo-simple

---

## Current Status

This repository currently contains **design, patches descriptions, and helpers**.  
Real modified Forgejo source code will be added once a build environment is available.

| Area                    | Status      |
|-------------------------|-------------|
| Vision & goals          | Done        |
| First change design     | Done        |
| Repository structure    | Done        |
| Real source patches     | Pending     |
| Buildable binary        | Pending     |

---

## Repository Structure

```
forgejo-simple/
├── README.md                  ← you are here
├── LICENSE
├── docs/
│   ├── SIMPLIFICATION.md      ← overall plan
│   ├── ROADMAP.md
│   └── NEXT.md
├── patches/
│   └── 0001-sqlite-only-minimal-install.md
├── examples/
│   └── minimal-app.ini
└── scripts/
    └── apply-minimal-install.sh
```

---

## The First Real Change (when we can build)

**Goal:** Turn the long Forgejo install page into a 4-field form.

- Force SQLite3 only
- Keep only: Instance title + Admin username + Password + Confirm
- Hide everything else behind good defaults

See `patches/0001-sqlite-only-minimal-install.md` for the exact planned edits.

---

## How to follow progress

Just say **Next** and we will complete the next small chunk.

---

Maintained by [@5mil](https://github.com/5mil)
