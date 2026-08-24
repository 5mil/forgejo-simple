# Next steps

## Done
- Basic HTTP server
- Repository list
- **Git smart HTTP** (via `git http-backend`)

You can now:

```bash
go run ./cmd/forgejo-simple

# in another terminal
git init --bare data/hello.git
git clone http://localhost:3000/git/hello.git
```

## Next
1. Simple repository view page (show README / basic info)
2. Optional create-repo endpoint
3. Basic authentication later if needed
