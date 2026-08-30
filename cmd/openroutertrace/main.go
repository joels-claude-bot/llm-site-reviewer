package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:9188", "address to listen on")
	upstream := flag.String("upstream", "https://openrouter.ai", "upstream origin")
	output := flag.String("output", "scratchpads/openrouter-wire.log", "wire log path")
	flag.Parse()

	target, err := url.Parse(*upstream)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.OpenFile(*output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var mu sync.Mutex
	write := func(direction string, body []byte) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(file, "\n=== %s ===\n", direction)
		var formatted bytes.Buffer
		if json.Indent(&formatted, body, "", "  ") == nil {
			_, _ = formatted.WriteTo(file)
		} else {
			_, _ = file.Write(body)
		}
		_, _ = file.WriteString("\n")
		_ = file.Sync()
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Body = &loggingBody{ReadCloser: resp.Body, write: func(chunk []byte) {
			write("OPENROUTER RESPONSE CHUNK", chunk)
		}}
		return nil
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		write("OPENCODE REQUEST "+req.Method+" "+req.URL.Path, body)
		proxy.ServeHTTP(w, req)
	})

	log.Printf("redacting trace proxy listening on http://%s", *listen)
	log.Fatal(http.ListenAndServe(*listen, handler))
}

type loggingBody struct {
	io.ReadCloser
	write func([]byte)
}

func (body *loggingBody) Read(p []byte) (int, error) {
	n, err := body.ReadCloser.Read(p)
	if n > 0 {
		chunk := append([]byte(nil), p[:n]...)
		body.write(chunk)
	}
	return n, err
}
