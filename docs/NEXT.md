# Current Status (Option 2 active)

Because the live build environment has been unavailable for a long time, we switched to writing complete ready-to-apply files.

## Files now in the repository

- `patches/install.tmpl.simplified`  
  Full minimal install page (only Title + Admin fields + hidden SQLite defaults)

- `patches/install.go.sqlite-only-snippet.md`  
  Exact Go changes needed in the install handler

## Next actions when environment returns
1. Clone Forgejo v16.0.3
2. Replace `templates/install.tmpl` with the simplified version
3. Apply the Go snippet
4. Build and test
5. Push the real source

Say **Next** to check the environment again, or continue refining the patches.
