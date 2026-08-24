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

type Commit struct {
	Hash    string
	Subject string
	Author  string
	Date    string
}

type TreeEntry struct {
	Mode string
	Type string // "blob" or "tree"
	Hash string
	Name string
}

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

	if _, err := exec.LookPath("git"); err != nil {
		log.Fatal("git binary not found in PATH – required for smart HTTP")
	}

	mux := http.NewServeMux()

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

	mux.HandleFunc("/repo/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/repo/")
		name = filepath.Clean(name)
		if name == "." || name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
			http.NotFound(w, r)
			return
		}

		repoPath := filepath.Join(absoluteData, name)
		if !isGitRepo(repoPath) {
			http.NotFound(w, r)
			return
		}

		renderRepo(w, name, repoPath)
	})

	mux.Handle("/git/", gitSmartHTTP(absoluteData))

	log.Printf("forgejo-simple listening on %s", *addr)
	log.Printf("  data directory : %s", absoluteData)
	log.Printf("  clone example  : git clone http://localhost%s/git/myproject.git", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

func gitSmartHTTP(projectRoot string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if isGitRepo(filepath.Join(dataDir, name)) {
			repos = append(repos, name)
		}
	}
	return repos, nil
}

func isGitRepo(path string) bool {
	return fileExists(filepath.Join(path, "HEAD")) || fileExists(filepath.Join(path, ".git"))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func recentCommits(repoPath string, n int) []Commit {
	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", n), "--pretty=format:%h%x09%s%x09%an%x09%ad", "--date=short")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    parts[0],
			Subject: parts[1],
			Author:  parts[2],
			Date:    parts[3],
		})
	}
	return commits
}

func listTree(repoPath string) []TreeEntry {
	// git ls-tree HEAD
	cmd := exec.Command("git", "ls-tree", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var entries []TreeEntry
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		// format: <mode> <type> <hash>\t<name>
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		meta := strings.Fields(parts[0])
		if len(meta) < 3 {
			continue
		}
		entries = append(entries, TreeEntry{
			Mode: meta[0],
			Type: meta[1],
			Hash: meta[2],
			Name: parts[1],
		})
	}
	return entries
}

func renderHome(w http.ResponseWriter, dataDir string, repos []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("home").Parse(homeTmpl))
	data := struct {
		DataDir string
		Repos  []string
	}{dataDir, repos}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
	}
}

func renderRepo(w http.ResponseWriter, name, repoPath string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	commits := recentCommits(repoPath, 10)
	tree := listTree(repoPath)
	cloneURL := fmt.Sprintf("/git/%s", name)

	tmpl := template.Must(template.New("repo").Parse(repoTmpl))
	data := struct {
		Name     string
		CloneURL string
		RepoPath string
		Commits  []Commit
		Tree     []TreeEntry
	}{
		Name:     name,
		CloneURL: cloneURL,
		RepoPath: repoPath,
		Commits:  commits,
		Tree:     tree,
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
	}
}

const homeTmpl = `<!DOCTYPE html>
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
    li { padding: 0.6rem 0; border-bottom: 1px solid #eee; }
    a { color: #0969da; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .empty { color: #666; }
    code { background: #f6f8fa; padding: 0.15em 0.4em; border-radius: 4px; font-size: 0.85em; }
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
</html>`

const repoTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Name}} · forgejo-simple</title>
  <style>
    :root { font-family: system-ui, -apple-system, sans-serif; }
    body { max-width: 42rem; margin: 2rem auto; padding: 0 1rem; color: #222; }
    h1 { font-size: 1.4rem; margin-bottom: 0.25rem; }
    h2 { font-size: 1.1rem; margin-top: 2rem; margin-bottom: 0.5rem; }
    .meta { color: #666; font-size: 0.9rem; margin-bottom: 1.5rem; }
    a { color: #0969da; text-decoration: none; }
    a:hover { text-decoration: underline; }
    code { background: #f6f8fa; padding: 0.15em 0.4em; border-radius: 4px; font-size: 0.9em; }
    .box { background: #f6f8fa; border: 1px solid #d0d7de; border-radius: 6px; padding: 0.8rem 1rem; margin: 1rem 0; }
    .back { font-size: 0.9rem; margin-bottom: 1rem; display: inline-block; }
    table { width: 100%; border-collapse: collapse; font-size: 0.9rem; }
    td { padding: 0.4rem 0.5rem 0.4rem 0; border-bottom: 1px solid #eee; vertical-align: top; }
    td.hash { width: 5.5rem; font-family: ui-monospace, monospace; color: #666; }
    td.date { width: 6.5rem; color: #666; white-space: nowrap; }
    td.mode { width: 4.5rem; font-family: ui-monospace, monospace; color: #666; }
    .empty { color: #666; }
    .dir { font-weight: 500; }
  </style>
</head>
<body>
  <a class="back" href="/">← all repositories</a>
  <h1>{{.Name}}</h1>

  <div class="box">
    <strong>Clone</strong><br>
    <code>git clone http://localhost:3000{{.CloneURL}}</code>
  </div>

  <h2>Files</h2>
  {{if .Tree}}
  <table>
    {{range .Tree}}
    <tr>
      <td class="mode">{{if eq .Type "tree"}}dir{{else}}file{{end}}</td>
      <td class="{{if eq .Type "tree"}}dir{{end}}">{{.Name}}</td>
    </tr>
    {{end}}
  </table>
  {{else}}
  <p class="empty">No files at HEAD (empty repository?).</p>
  {{end}}

  <h2>Recent commits</h2>
  {{if .Commits}}
  <table>
    {{range .Commits}}
    <tr>
      <td class="hash">{{.Hash}}</td>
      <td>{{.Subject}}</td>
      <td class="date">{{.Date}}</td>
    </tr>
    {{end}}
  </table>
  {{else}}
  <p class="empty">No commits yet.</p>
  {{end}}

  <p class="meta" style="margin-top:2rem">Path on disk: <code>{{.RepoPath}}</code></p>
</body>
</html>`
