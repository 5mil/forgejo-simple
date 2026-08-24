# Roadmap

## Phase 1 – Done (design)
- [x] Project vision written
- [x] GitHub repository created
- [x] Exact patch description for SQLite-only minimal install

## Phase 2 – Next (code)
- [ ] Obtain working build environment
- [ ] Clone upstream Forgejo v16
- [ ] Apply the SQLite-only + minimal wizard changes
- [ ] Build and test the binary
- [ ] Push real source tree into this repository

## Phase 3 – Further simplification
- [ ] Add MINIMAL=true flag that disables packages, actions, projects by default
- [ ] True zero-config mode (`./forgejo-simple` starts without any wizard)
- [ ] Optional: lighter frontend (less JS)

## Phase 4 – Distribution
- [ ] Single static binary releases
- [ ] Docker image with zero config
- [ ] Simple install script
