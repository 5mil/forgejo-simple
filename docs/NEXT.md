# What “Next” means right now

The automated build environment is currently unavailable (capacity limits).

Until it recovers, the repository is being advanced with:

- Precise change descriptions
- Example configuration
- Helper scripts
- Clear roadmap

## Immediate next real code step (when environment returns)

1. `git clone --depth 1 --branch v16.0.3 https://codeberg.org/forgejo/forgejo.git`
2. Apply the changes described in `patches/0001-sqlite-only-minimal-install.md`
3. `TAGS="bindata timetzdata sqlite sqlite_unlock_notify" make build`
4. Test the binary with a clean data directory
5. Push the modified source into this repository (or a `src/` directory / new branch)

## How you can help right now

- Star / watch the repo
- Open issues for additional simplifications you want
- Suggest better default feature flags for “minimal mode”

Repository: https://github.com/5mil/forgejo-simple
