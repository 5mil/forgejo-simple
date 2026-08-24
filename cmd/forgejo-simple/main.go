package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", ":3000", "HTTP listen address")
	dataDir := flag.String("data", "./data", "Directory that holds Git repositories")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("cannot create data directory: %v", err)
	}

	mux := http.NewServeMux()

	// Placeholder root handler – will become the repo list
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>forgejo-simple</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 40rem; margin: 2rem auto; padding: 0 1rem; }
    h1 { font-size: 1.5rem; }
    code { background: #f4f4f4; padding: 0.1em 0.3em; border-radius: 3px; }
  </style>
</head>
<body>
  <h1>forgejo-simple</h1>
  <p>Minimal Git forge is running.</p>
  <p>Data directory: <code>%s</code></p>
  <p>Next steps: add Git smart HTTP and a real repository list.</p>
</body>
</html>`, *dataDir)
	})

	log.Printf("forgejo-simple listening on %s (data=%s)", *addr, *dataDir)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
