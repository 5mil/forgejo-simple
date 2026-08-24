package main

import (
	"flag"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
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
	Type string
	Hash string
	Name string
}

type Crumb struct {
	Name string
	Href string
}

var validRepoName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

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
		log.Fatal("git binary not found in PATH")
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

	mux.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			renderNew(w, "")
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				renderNew(w, "invalid form")
				return
			}
			name := strings.TrimSpace(r.FormValue("name"))
			if name == "" {
				renderNew(w, "name is required")
				return
			}
			name = strings.TrimSuffix(name, ".git")
			if !validRepoName.MatchString(name) {
				renderNew(w, "name may only contain letters, numbers, dots, underscores and hyphens")
				return
			}
			repoPath := filepath.Join(absoluteData, name+".git")
			if isGitRepo(repoPath) || isGitRepo(filepath.Join(absoluteData, name)) {
				renderNew(w, "repository already exists")
				return
			}
			if out, err := exec.Command("git", "init", "--bare", repoPath).CombinedOutput(); err != nil {
				log.Printf("git init failed: %v %s", err, out)
				renderNew(w, "failed to create repository")
				return
			}
			http.Redirect(w, r, "/repo/"+name+".git", http.StatusSeeOther)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
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

	log.Printf("forgejo-simple listening on %s (data=%s)", *addr, absoluteData)
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
		(&cgi.Handler{
			Path: "git",
			Args: []string{"http-backend"},
			Env:  []string{"GIT_PROJECT_ROOT=" + projectRoot, "GIT_HTTP_EXPORT_ALL=1", "PATH_INFO=" + pathInfo},
			Dir:  projectRoot,
		}).ServeHTTP(w, r)
	})
}

func listRepos(dataDir string) ([]string, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}
	var repos []string
	for _, e := range entries {
		if e.IsDir() && isGitRepo(filepath.Join(dataDir, e.Name())) {
			repos = append(repos, e.Name())
		}
	}
	return repos, nil
}

func isGitRepo(p string) bool {
	return fileExists(filepath.Join(p, "HEAD")) || fileExists(filepath.Join(p, ".git"))
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
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
		if len(parts) >= 4 {
			commits = append(commits, Commit{Hash: parts[0], Subject: parts[1], Author: parts[2], Date: parts[3]})
		}
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
		if len(meta) >= 3 {
			entries = append(entries, TreeEntry{Mode: meta[0], Type: meta[1], Hash: meta[2], Name: path.Base(parts[1])})
		}
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

func tryReadme(repoPath string) (string, string) {
	for _, name := range []string{"README.md", "Readme.md", "readme.md", "README", "README.txt"} {
		if content, err := getBlob(repoPath, name); err == nil {
			return name, content
		}
	}
	return "", ""
}

func breadcrumbs(repoName, filePath, kind string) []Crumb {
	crumbs := []Crumb{{Name: repoName, Href: "/repo/" + repoName}}
	if filePath == "" {
		return crumbs
	}
	parts := strings.Split(filePath, "/")
	acc := ""
	for i, p := range parts {
		if acc == "" {
			acc = p
		} else {
			acc += "/" + p
		}
		isLast := i == len(parts)-1
		if isLast {
			crumbs = append(crumbs, Crumb{Name: p, Href: ""})
		} else {
			crumbs = append(crumbs, Crumb{Name: p, Href: "/repo/" + repoName + "/tree/" + acc})
		}
	}
	return crumbs
}

// renderMarkdown is a tiny, dependency-free Markdown subset renderer.
// Supports: # headings, **bold**, *italic*, `code`, ``` fences, - lists, [links](url), paragraphs.
func renderMarkdown(src string) template.HTML {
	lines := strings.Split(src, "\n")
	var out strings.Builder
	inFence := false
	inList := false

	flushList := func() {
		if inList {
			out.WriteString("</ul>\n")
			inList = false
		}
	}

	inline := func(s string) string {
		s = html.EscapeString(s)
		// code
		s = regexp.MustCompile(`\`+"`"+`([^`+"`"+`]+)\`+"`"+``).ReplaceAllString(s, "<code>$1</code>")
		// bold
		s = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(s, "<strong>$1</strong>")
		// italic
		s = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(s, "<em>$1</em>")
		// links
		s = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllString(s, `<a href="$2">$1</a>`)
		return s
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// fenced code
		if strings.HasPrefix(trimmed, "```") {
			flushList()
			if inFence {
				out.WriteString("</code></pre>\n")
				inFence = false
			} else {
				out.WriteString("<pre><code>")
				inFence = true
			}
			continue
		}
		if inFence {
			out.WriteString(html.EscapeString(line) + "\n")
			continue
		}

		// headings
		if m := regexp.MustCompile(`^(#{1,6})\s+(.*)$`).FindStringSubmatch(trimmed); m != nil {
			flushList()
			level := len(m[1])
			fmt.Fprintf(&out, "<h%d>%s</h%d>\n", level, inline(m[2]), level)
			continue
		}

		// unordered list
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			if !inList {
				out.WriteString("<ul>\n")
				inList = true
			}
			fmt.Fprintf(&out, "<li>%s</li>\n", inline(trimmed[2:]))
			continue
		}

		flushList()

		if trimmed == "" {
			out.WriteString("\n")
			continue
		}

		fmt.Fprintf(&out, "<p>%s</p>\n", inline(trimmed))
	}
	flushList()
	if inFence {
		out.WriteString("</code></pre>\n")
	}
	return template.HTML(out.String())
}

func renderHome(w http.ResponseWriter, dataDir string, repos []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	template.Must(template.New("home").Parse(homeTmpl)).Execute(w, struct {
		DataDir string
		Repos  []string
	}{dataDir, repos})
}

func renderNew(w http.ResponseWriter, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	template.Must(template.New("new").Parse(newTmpl)).Execute(w, struct{ Error string }{errMsg})
}

func renderRepo(w http.ResponseWriter, name, repoPath string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	readmeName, readmeRaw := tryReadme(repoPath)
	var readmeHTML template.HTML
	if readmeName != "" && strings.HasSuffix(strings.ToLower(readmeName), ".md") {
		readmeHTML = renderMarkdown(readmeRaw)
	} else if readmeRaw != "" {
		readmeHTML = template.HTML("<pre>" + html.EscapeString(readmeRaw) + "</pre>")
	}
	template.Must(template.New("repo").Parse(repoTmpl)).Execute(w, struct {
		Name, CloneURL, RepoPath, ReadmeName string
		ReadmeHTML                           template.HTML
		Commits                              []Commit
		Tree                                 []TreeEntry
	}{
		Name: name, CloneURL: "/git/" + name, RepoPath: repoPath,
		ReadmeName: readmeName, ReadmeHTML: readmeHTML,
		Commits: recentCommits(repoPath, 10), Tree: listTree(repoPath, ""),
	})
}

func renderTree(w http.ResponseWriter, repoName, repoPath, treePath string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	template.Must(template.New("tree").Parse(treeTmpl)).Execute(w, struct {
		RepoName, TreePath string
		Tree               []TreeEntry
		Crumbs             []Crumb
	}{repoName, treePath, listTree(repoPath, treePath), breadcrumbs(repoName, treePath, "tree")})
}

func renderBlob(w http.ResponseWriter, repoName, repoPath, filePath string) {
	content, err := getBlob(repoPath, filePath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	template.Must(template.New("blob").Parse(blobTmpl)).Execute(w, struct {
		RepoName, FilePath, Content string
		Crumbs                      []Crumb
	}{repoName, filePath, content, breadcrumbs(repoName, filePath, "blob")})
}

const homeTmpl = `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>forgejo-simple</title>
<style>
:root{font-family:system-ui,-apple-system,sans-serif}
body{max-width:42rem;margin:2rem auto;padding:0 1rem;color:#222}
h1{font-size:1.4rem;margin-bottom:.25rem}
.meta{color:#666;font-size:.9rem;margin-bottom:1.5rem}
ul{list-style:none;padding:0}li{padding:.6rem 0;border-bottom:1px solid #eee}
a{color:#0969da;text-decoration:none}a:hover{text-decoration:underline}
.empty{color:#666}code{background:#f6f8fa;padding:.15em .4em;border-radius:4px;font-size:.85em}
.actions{margin-bottom:1.5rem}
.btn{display:inline-block;background:#0969da;color:#fff;padding:.4rem .8rem;border-radius:6px;font-size:.9rem;text-decoration:none}
.btn:hover{background:#0550ae;color:#fff;text-decoration:none}
</style></head><body>
<h1>forgejo-simple</h1>
<p class="meta">Minimal Git forge &middot; data: <code>{{.DataDir}}</code></p>
<div class="actions"><a class="btn" href="/new">New repository</a></div>
{{if .Repos}}<ul>{{range .Repos}}<li><a href="/repo/{{.}}">{{.}}</a></li>{{end}}</ul>
{{else}}<p class="empty">No repositories found.</p>
<p class="empty">Create one with the button above.</p>{{end}}
</body></html>`

const newTmpl = `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>New repository</title>
<style>
:root{font-family:system-ui,-apple-system,sans-serif}
body{max-width:28rem;margin:2rem auto;padding:0 1rem;color:#222}
h1{font-size:1.3rem}.back{font-size:.9rem;margin-bottom:1rem;display:inline-block}
a{color:#0969da;text-decoration:none}a:hover{text-decoration:underline}
label{display:block;margin-bottom:.3rem;font-size:.9rem}
input[type=text]{width:100%;padding:.5rem .6rem;border:1px solid #d0d7de;border-radius:6px;font-size:1rem;box-sizing:border-box}
.hint{font-size:.8rem;color:#666;margin-top:.3rem}.error{color:#cf222e;font-size:.9rem;margin-bottom:1rem}
button{margin-top:1rem;background:#0969da;color:#fff;border:none;padding:.5rem 1rem;border-radius:6px;font-size:.95rem;cursor:pointer}
button:hover{background:#0550ae}
</style></head><body>
<a class="back" href="/">← all repositories</a><h1>New repository</h1>
{{if .Error}}<p class="error">{{.Error}}</p>{{end}}
<form method="post" action="/new">
<label for="name">Repository name</label>
<input type="text" id="name" name="name" required autofocus pattern="[a-zA-Z0-9._-]+" placeholder="myproject">
<p class="hint">Letters, numbers, dots, underscores, hyphens.</p>
<button type="submit">Create</button></form></body></html>`

const repoTmpl = `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Name}} · forgejo-simple</title>
<style>
:root{font-family:system-ui,-apple-system,sans-serif}
body{max-width:42rem;margin:2rem auto;padding:0 1rem;color:#222}
h1{font-size:1.4rem;margin-bottom:.25rem}h2{font-size:1.1rem;margin-top:2rem;margin-bottom:.5rem}
.meta{color:#666;font-size:.9rem}
a{color:#0969da;text-decoration:none}a:hover{text-decoration:underline}
code{background:#f6f8fa;padding:.15em .4em;border-radius:4px;font-size:.9em}
.box{background:#f6f8fa;border:1px solid #d0d7de;border-radius:6px;padding:.8rem 1rem;margin:1rem 0}
.back{font-size:.9rem;margin-bottom:1rem;display:inline-block}
table{width:100%;border-collapse:collapse;font-size:.9rem}
td{padding:.4rem .5rem .4rem 0;border-bottom:1px solid #eee;vertical-align:top}
td.hash{width:5.5rem;font-family:ui-monospace,monospace;color:#666}
td.date{width:6.5rem;color:#666;white-space:nowrap}
td.mode{width:4.5rem;font-family:ui-monospace,monospace;color:#666}
.empty{color:#666}.dir{font-weight:500}
.readme{border:1px solid #d0d7de;border-radius:6px;padding:1rem 1.2rem;margin-top:1.5rem;font-size:.95rem;line-height:1.55}
.readme h1,.readme h2,.readme h3{margin-top:1.2em;margin-bottom:.4em}
.readme h1{font-size:1.3rem}.readme h2{font-size:1.15rem}.readme h3{font-size:1.05rem}
.readme p{margin:.6em 0}.readme ul{padding-left:1.4rem;margin:.6em 0}
.readme pre{background:#f6f8fa;padding:.8rem;border-radius:4px;overflow:auto;font-size:.85rem}
.readme code{background:#f6f8fa;padding:.1em .3em;border-radius:3px;font-size:.85em}
.readme-title{font-size:.85rem;color:#666;margin-bottom:.4rem}
</style></head><body>
<a class="back" href="/">← all repositories</a>
<h1>{{.Name}}</h1>
<div class="box"><strong>Clone</strong><br><code>git clone http://localhost:3000{{.CloneURL}}</code></div>
{{if .ReadmeHTML}}<div class="readme-title">{{.ReadmeName}}</div><div class="readme">{{.ReadmeHTML}}</div>{{end}}
<h2>Files</h2>
{{if .Tree}}<table>{{range .Tree}}<tr>
<td class="mode">{{if eq .Type "tree"}}dir{{else}}file{{end}}</td>
<td class="{{if eq .Type "tree"}}dir{{end}}">
{{if eq .Type "blob"}}<a href="/repo/{{$.Name}}/blob/{{.Name}}">{{.Name}}</a>
{{else}}<a href="/repo/{{$.Name}}/tree/{{.Name}}">{{.Name}}</a>{{end}}
</td></tr>{{end}}</table>
{{else}}<p class="empty">No files at HEAD.</p>{{end}}
<h2>Recent commits</h2>
{{if .Commits}}<table>{{range .Commits}}<tr>
<td class="hash">{{.Hash}}</td><td>{{.Subject}}</td><td class="date">{{.Date}}</td>
</tr>{{end}}</table>{{else}}<p class="empty">No commits yet.</p>{{end}}
<p class="meta" style="margin-top:2rem">Path: <code>{{.RepoPath}}</code></p>
</body></html>`

const treeTmpl = `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.TreePath}} · {{.RepoName}}</title>
<style>
:root{font-family:system-ui,-apple-system,sans-serif}
body{max-width:42rem;margin:2rem auto;padding:0 1rem;color:#222}
h1{font-size:1.2rem;margin-bottom:.5rem;word-break:break-all}
.crumbs{font-size:.9rem;margin-bottom:1rem;color:#666}
.crumbs a{color:#0969da;text-decoration:none}.crumbs a:hover{text-decoration:underline}
.crumbs span{margin:0 .25rem}
table{width:100%;border-collapse:collapse;font-size:.9rem}
td{padding:.4rem .5rem .4rem 0;border-bottom:1px solid #eee}
td.mode{width:4.5rem;font-family:ui-monospace,monospace;color:#666}
.dir{font-weight:500}.empty{color:#666}
a{color:#0969da;text-decoration:none}a:hover{text-decoration:underline}
</style></head><body>
<nav class="crumbs">{{range $i,$c := .Crumbs}}{{if $i}}<span>/</span>{{end}}{{if $c.Href}}<a href="{{$c.Href}}">{{$c.Name}}</a>{{else}}<strong>{{$c.Name}}</strong>{{end}}{{end}}</nav>
<h1>{{if .TreePath}}{{.TreePath}}{{else}}/{{end}}</h1>
{{if .Tree}}<table>{{range .Tree}}<tr>
<td class="mode">{{if eq .Type "tree"}}dir{{else}}file{{end}}</td>
<td class="{{if eq .Type "tree"}}dir{{end}}">
{{if eq .Type "blob"}}<a href="/repo/{{$.RepoName}}/blob/{{if $.TreePath}}{{$.TreePath}}/{{end}}{{.Name}}">{{.Name}}</a>
{{else}}<a href="/repo/{{$.RepoName}}/tree/{{if $.TreePath}}{{$.TreePath}}/{{end}}{{.Name}}">{{.Name}}</a>{{end}}
</td></tr>{{end}}</table>{{else}}<p class="empty">Empty directory.</p>{{end}}
</body></html>`

const blobTmpl = `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.FilePath}} · {{.RepoName}}</title>
<style>
:root{font-family:system-ui,-apple-system,sans-serif}
body{max-width:50rem;margin:2rem auto;padding:0 1rem;color:#222}
h1{font-size:1.2rem;margin-bottom:.5rem;word-break:break-all}
.crumbs{font-size:.9rem;margin-bottom:1rem;color:#666}
.crumbs a{color:#0969da;text-decoration:none}.crumbs a:hover{text-decoration:underline}
.crumbs span{margin:0 .25rem}
pre{background:#f6f8fa;border:1px solid #d0d7de;border-radius:6px;padding:1rem;overflow:auto;font-size:.85rem;line-height:1.45}
</style></head><body>
<nav class="crumbs">{{range $i,$c := .Crumbs}}{{if $i}}<span>/</span>{{end}}{{if $c.Href}}<a href="{{$c.Href}}">{{$c.Name}}</a>{{else}}<strong>{{$c.Name}}</strong>{{end}}{{end}}</nav>
<h1>{{.FilePath}}</h1>
<pre>{{.Content}}</pre>
</body></html>`
