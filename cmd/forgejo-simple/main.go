package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	addr := flag.String("addr", ":3000", "HTTP listen address")
	dataDir := flag.String("data", "./data", "Directory that holds Git repositories")
	flag.Parse()

	absoluteData, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatalf("data directory: %v", err)
	}
	if err := os.MkdirAll(absoluteData, 0o755); err != nil {
		log.Fatalf("cannot create data directory: %v", err)
	}

	// Make sure git is available
	if _, err := exec.LookPath("git"); err != nil {
		log.Fatal("git binary not found in PATH – required for smart HTTP")
	}

	mux := http.NewServeMux()

	// Home page – list repositories
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		repos, err := listRepos(absoluteData)
		if err != nil {
			http.Error(w, "failed to list repositories", http.StatusInternalServerError)
			return
		}
		renderHome(w, absoluteData, repos)
	})

	// Git smart HTTP – any path under /git/ is handled by git http-backend
	// Example clone URL: http://localhost:3000/git/myproject.git
	mux.Handle("/git/", gitSmartHTTP(absoluteData))

	log.Printf("forgejo-simple listening on %s", *addr)
	log.Printf("  data directory : %s", absoluteData)
	log.Printf("  clone example  : git clone http://localhost%s/git/myproject.git", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

// gitSmartHTTP returns a handler that uses git http-backend (CGI).
func gitSmartHTTP(projectRoot string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Translate /git/myrepo.git/... → PATH_INFO=/myrepo.git/...
		pathInfo := strings.TrimPrefix(r.URL.Path, "/git")
		if pathInfo == "" {
			pathInfo = "/"
		}

		handler := &cgi.Handler{
			Path: "git",
			Args: []string{"http-backend"},
			Env: []string{
				"GIT_PROJECT_ROOT=" + projectRoot,
				"GIT_HTTP_EXPORT_ALL=1",
				"PATH_INFO=" + pathInfo,
			},
			Dir: projectRoot,
		}
		handler.ServeHTTP(w, r)
	})
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
    li { padding: 0.6rem 0; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; align-items: center; }
    a { color: #0969da; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .empty { color: #666; }
    code { background: #f6f8fa; padding: 0.15em 0.4em; border-radius: 4px; font-size: 0.85em; }
    .clone { font-size: 0.8rem; color: #666; }
  </style>
</head>
<body>
  <h1>forgejo-simple</h1>
  <p class="meta">Minimal Git forge &middot; data: <code>{{.DataDir}}</code></p>

  {{if .Repos}}
  <ul>
    {{range .Repos}}
    <li>
      <span>{{.}}</span>
      <span class="clone"><code>git clone /git/{{.}}</code></span>
    </li>
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
