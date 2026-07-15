// Command testsite serves one folder of markdown as a tiny rendered site.
// It is Go's minimal answer to `mkdocs serve`: render each .md with goldmark on
// request and serve everything else (images, svg) verbatim. Used to preview the
// testie/ demo corpus, a worked "problem" example with deliberately planted defects.
//
//	go run ./cmd/testsite                       # serve ./testie on :9698
//	go run ./cmd/testsite -dir x -port 8080     # any folder, any port
//
//arch:tool
package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// page wraps rendered markdown in just enough HTML to be readable in a browser.
var page = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
 body{max-width:44rem;margin:2rem auto;padding:0 1rem;font:16px/1.6 system-ui,sans-serif;color:#1a1a1a}
 img{max-width:100%;height:auto}
 table{border-collapse:collapse}th,td{border:1px solid #ccc;padding:.4rem .6rem}
 code{background:#f4f4f4;padding:.1rem .3rem;border-radius:3px}
 a{color:#0b7285}
</style>
</head>
<body>{{.Body}}</body>
</html>`))

func main() {
	var dir, port string
	flag.StringVar(&dir, "dir", "testie", "directory of markdown to serve")
	flag.StringVar(&port, "port", "9698", "port to listen on")
	flag.Parse()

	root, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("resolve dir: %v", err)
	}
	markdown := goldmark.New(goldmark.WithExtensions(extension.GFM))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Map the URL to a file inside root. Cleaning a rooted path drops any
		// leading .. so a request can never escape the served folder.
		rel := strings.TrimPrefix(filepath.Clean("/"+r.URL.Path), "/")
		switch {
		case rel == "":
			rel = "index.md"
		case filepath.Ext(rel) == "":
			rel += ".md" // pretty links: /spec resolves to spec.md
		}
		full := filepath.Join(root, rel)

		// Anything that is not markdown (svg, png, css) is served as-is.
		if filepath.Ext(full) != ".md" {
			http.ServeFile(w, r, full)
			return
		}

		src, err := os.ReadFile(full)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		var body strings.Builder
		if err := markdown.Convert(src, &body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data := struct {
			Title string
			Body  template.HTML
		}{Title: strings.TrimSuffix(rel, ".md"), Body: template.HTML(body.String())}
		if err := page.Execute(w, data); err != nil {
			log.Printf("render %s: %v", rel, err)
		}
	})

	log.Printf("serving %s on http://localhost:%s", root, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
