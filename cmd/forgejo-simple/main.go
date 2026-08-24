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
	"path"
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
		rest := strings.TrimPrefix(r.URL.Path, "/repo/")
		parts := strings.SplitN(rest, "/", 3)

		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}

		name := filepath.Clean(parts[0])
		if name == "." || strings.Contains(name, "..") {
			http.NotFound(w, r)
			return
		}

		repoPath := filepath.Join(absoluteData, name)
		if !isGitRepo(repoPath) {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 1 {
			renderRepo(w, name, repoPath)
			return
		}

		sub := parts[1]
		filePath := ""
		if len(parts) == 3 {
			filePath = parts[2]
		}
		filePath = path.Clean("/" + filePath)
		filePath = strings.TrimPrefix(filePath, "/")
		if strings.Contains(filePath, "..") {
			http.NotFound(w, r)
			return
		}

		switch sub {
		case "blob":
			renderBlob(w, name, repoPath, filePath)
		case "tree":
			renderTree(w, name, repoPath, filePath)
		default:
			http.NotFound(w, r)
		}
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

func listTree(repoPath, treePath string) []TreeEntry {
	args := []string{"ls-tree", "HEAD"}
	if treePath != "" {
		args = append(args, treePath+"/")
	}
	cmd := exec.Command("git", args...)
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
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		meta := strings.Fields(parts[0])
		if len(meta) < 3 {
			continue
		}
		name := path.Base(parts[1])
		entries = append(entries, TreeEntry{
			Mode: meta[0],
			Type: meta[1],
			Hash: meta[2],
			Name: name,
		})
	}
	return entries
}

func getBlob(repoPath, filePath string) (string, error) {
	cmd := exec.Command("git", "show", "HEAD:"+filePath)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// tryReadme looks for common README filenames at the repository root.
func tryReadme(repoPath string) (string, string) {
	candidates := []string{"README.md", "Readme.md", "readme.md", "README", "README.txt"}
	for _, name := range candidates {
		content, err := getBlob(repoPath, name)
		if err == nil {
			return name, content
		}
	}
	return "", ""
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
	tree := listTree(repoPath, "")
	readmeName, readmeContent := tryReadme(repoPath)
	cloneURL := fmt.Sprintf("/git/%s", name)

	tmpl := template.Must(template.New("repo").Parse(repoTmpl))
	data := struct {
		Name          string
		CloneURL      string
		RepoPath      string
		Commits       []Commit
		Tree          []TreeEntry
		ReadmeName    string
		ReadmeContent string
	}{
		Name:          name,
		CloneURL:      cloneURL,
		RepoPath:      repoPath,
		Commits:       commits,
		Tree:          tree,
		ReadmeName:    readmeName,
		ReadmeContent: readmeContent,
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
	}
}

func renderTree(w http.ResponseWriter, repoName, repoPath, treePath string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tree := listTree(repoPath, treePath)

	tmpl := template.Must(template.New("tree").Parse(treeTmpl))
	data := struct {
		RepoName string
		TreePath string
		Tree     []TreeEntry
		Parent   string
	}{
		RepoName: repoName,
		TreePath: treePath,
		Tree:     tree,
		Parent:   path.Dir(treePath),
	}
	if data.Parent == "." {
		data.Parent = ""
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
	}
}

func renderBlob(w http.ResponseWriter, repoName, repoPath, filePath string) {
	content, err := getBlob(repoPath, filePath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("blob").Parse(blobTmpl))
	data := struct {
		RepoName string
		FilePath string
		Content  string
		Parent   string
	}{
		RepoName: repoName,
		FilePath: filePath,
		Content:  content,
		Parent:   path.Dir(filePath),
	}
	if data.Parent == "." {
		data.Parent = ""
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
    .readme {
      background: #f6f8fa;
      border: 1px solid #d0d7de;
      border-radius: 6px;
      padding: 1rem 1.2rem;
      margin-top: 1.5rem;
      white-space: pre-wrap;
      font-size: 0.9rem;
      line-height: 1.5;
    }
    .readme-title { font-size: 0.85rem; color: #666; margin-bottom: 0.5rem; }
  </style>
</head>
<body>
  <a class="back" href="/">← all repositories</a>
  <h1>{{.Name}}</h1>

  <div class="box">
    <strong>Clone</strong><br>
    <code>git clone http://localhost:3000{{.CloneURL}}</code>
  </div>

  {{if .ReadmeContent}}
  <div class="readme-title">{{.ReadmeName}}</div>
  <div class="readme">{{.ReadmeContent}}</div>
  {{end}}

  <h2>Files</h2>
  {{if .Tree}}
  <table>
    {{range .Tree}}
    <tr>
      <td class="mode">{{if eq .Type "tree"}}dir{{else}}file{{end}}</td>
      <td class="{{if eq .Type "tree"}}dir{{end}}">
        {{if eq .Type "blob"}}
          <a href="/repo/{{$.Name}}/blob/{{.Name}}">{{.Name}}</a>
        {{else}}
          <a href="/repo/{{$.Name}}/tree/{{.Name}}">{{.Name}}</a>
        {{end}}
      </td>
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

const treeTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.TreePath}} · {{.RepoName}}</title>
  <style>
    :root { font-family: system-ui, -apple-system, sans-serif; }
    body { max-width: 42rem; margin: 2rem auto; padding: 0 1rem; color: #222; }
    h1 { font-size: 1.2rem; margin-bottom: 0.5rem; word-break: break-all; }
    .back { font-size: 0.9rem; margin-bottom: 1rem; display: inline-block; }
    a { color: #0969da; text-decoration: none; }
    a:hover { text-decoration: underline; }
    table { width: 100%; border-collapse: collapse; font-size: 0.9rem; }
    td { padding: 0.4rem 0.5rem 0.4rem 0; border-bottom: 1px solid #eee; }
    td.mode { width: 4.5rem; font-family: ui-monospace, monospace; color: #666; }
    .dir { font-weight: 500; }
    .empty { color: #666; }
  </style>
</head>
<body>
  {{if .TreePath}}
    {{if .Parent}}
      <a class="back" href="/repo/{{.RepoName}}/tree/{{.Parent}}">← {{.Parent}}</a>
    {{else}}
      <a class="back" href="/repo/{{.RepoName}}">← {{.RepoName}}</a>
    {{end}}
  {{else}}
    <a class="back" href="/repo/{{.RepoName}}">← {{.RepoName}}</a>
  {{end}}

  <h1>{{if .TreePath}}{{.TreePath}}{{else}}/{{end}}</h1>

  {{if .Tree}}
  <table>
    {{range .Tree}}
    <tr>
      <td class="mode">{{if eq .Type "tree"}}dir{{else}}file{{end}}</td>
      <td class="{{if eq .Type "tree"}}dir{{end}}">
        {{if eq .Type "blob"}}
          <a href="/repo/{{$.RepoName}}/blob/{{if $.TreePath}}{{$.TreePath}}/{{end}}{{.Name}}">{{.Name}}</a>
        {{else}}
          <a href="/repo/{{$.RepoName}}/tree/{{if $.TreePath}}{{$.TreePath}}/{{end}}{{.Name}}">{{.Name}}</a>
        {{end}}
      </td>
    </tr>
    {{end}}
  </table>
  {{else}}
  <p class="empty">Empty directory.</p>
  {{end}}
</body>
</html>`

const blobTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.FilePath}} · {{.RepoName}}</title>
  <style>
    :root { font-family: system-ui, -apple-system, sans-serif; }
    body { max-width: 50rem; margin: 2rem auto; padding: 0 1rem; color: #222; }
    h1 { font-size: 1.2rem; margin-bottom: 0.5rem; word-break: break-all; }
    .back { font-size: 0.9rem; margin-bottom: 1rem; display: inline-block; }
    a { color: #0969da; text-decoration: none; }
    a:hover { text-decoration: underline; }
    pre {
      background: #f6f8fa;
      border: 1px solid #d0d7de;
      border-radius: 6px;
      padding: 1rem;
      overflow: auto;
      font-size: 0.85rem;
      line-height: 1.45;
    }
  </style>
</head>
<body>
  {{if .Parent}}
    <a class="back" href="/repo/{{.RepoName}}/tree/{{.Parent}}">← {{.Parent}}</a>
  {{else}}
    <a class="back" href="/repo/{{.RepoName}}">← {{.RepoName}}</a>
  {{end}}
  <h1>{{.FilePath}}</h1>
  <pre>{{.Content}}</pre>
</body>
</html>`
