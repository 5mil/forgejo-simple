package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	addr := flag.String("addr", ":3000", "HTTP listen address")
	dataDir := flag.String("data", "./data", "Directory that holds Git repositories")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("cannot create data directory: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		repos, err := listRepos(*dataDir)
		if err != nil {
			http.Error(w, "failed to list repositories", http.StatusInternalServerError)
			return
		}
		renderHome(w, *dataDir, repos)
	})

	log.Printf("forgejo-simple listening on %s (data=%s)", *addr, *dataDir)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

// listRepos returns the names of directories under dataDir that look like Git repositories.
func listRepos(dataDir string) ([]string, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}

	var repos []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Accept both bare (repo.git) and non-bare (repo/.git) repositories
		gitDir := filepath.Join(dataDir, name, ".git")
		bareGit := filepath.Join(dataDir, name, "HEAD")
		if fileExists(gitDir) || fileExists(bareGit) {
			repos = append(repos, name)
		}
	}
	return repos, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func renderHome(w http.ResponseWriter, dataDir string, repos []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl := template.Must(template.New("home").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>forgejo-simple</title>
  <style>
    :root { font-family: system-ui, -apple-system, sans-serif; }
    body { max-width: 42rem; margin: 2rem auto; padding: 0 1rem; color: #222; }
    h1 { font-size: 1.4rem; margin-bottom: 0.25rem; }
    .meta { color: #666; font-size: 0.9rem; margin-bottom: 1.5rem; }
    ul { list-style: none; padding: 0; }
    li { padding: 0.5rem 0; border-bottom: 1px solid #eee; }
    a { color: #0969da; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .empty { color: #666; }
    code { background: #f6f8fa; padding: 0.15em 0.4em; border-radius: 4px; font-size: 0.9em; }
  </style>
</head>
<body>
  <h1>forgejo-simple</h1>
  <p class="meta">Minimal Git forge &middot; data: <code>{{.DataDir}}</code></p>

  {{if .Repos}}
  <ul>
    {{range .Repos}}
    <li><a href="/repo/{{.}}">{{.}}</a></li>
    {{end}}
  </ul>
  {{else}}
  <p class="empty">No repositories found.</p>
  <p class="empty">Create a bare repo with:<br>
  <code>git init --bare {{.DataDir}}/myproject.git</code></p>
  {{end}}
</body>
</html>`))

	data := struct {
		DataDir string
		Repos  []string
	}{
		DataDir: dataDir,
		Repos:  repos,
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
	}
}

// ensure we don't accidentally treat paths with .. as valid later
func cleanRepoName(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSuffix(name, ".git")
	return name
}
