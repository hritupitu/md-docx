package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	webview "github.com/webview/webview_go"
)

//go:embed web
var webFiles embed.FS

type uploadedFile struct {
	name    string
	tmpPath string
}

type job struct {
	files  []uploadedFile
	refDoc string
}

var (
	jobs      sync.Map
	downloads sync.Map
	server    *http.Server
)

func main() {
	// Pick a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	// Start HTTP server in background
	mux := http.NewServeMux()
	webFS, _ := fs.Sub(webFiles, "web")
	mux.Handle("/", http.FileServer(http.FS(webFS)))
	mux.HandleFunc("/upload", handleUpload)
	mux.HandleFunc("/events", handleEvents)
	mux.HandleFunc("/download", handleDownload)
	mux.HandleFunc("/check", handleCheck)

	server = &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}
	go server.ListenAndServe()

	// Open native webview window on the main thread
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("md2docx")
	w.SetSize(660, 780, webview.HintNone)
	w.Navigate(url)
	w.Run() // blocks until window is closed

	// Window closed — clean up
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	jobs.Range(func(_, v any) bool {
		j := v.(*job)
		if len(j.files) > 0 {
			os.RemoveAll(filepath.Dir(j.files[0].tmpPath))
		}
		return true
	})
}

// ── dependency check ──────────────────────────────────────────────────────────

func handleCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if bin := findPandoc(); bin != "" {
		out, _ := exec.Command(bin, "--version").Output()
		ver := ""
		if len(out) > 0 {
			ver = strings.Fields(string(out))[1]
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "version": ver})
	} else {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "hint": installHint()})
	}
}

func installHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install pandoc"
	case "linux":
		return "sudo apt install pandoc"
	case "windows":
		return "winget install JohnMacFarlane.Pandoc"
	}
	return "https://pandoc.org/installing.html"
}

// ── upload ────────────────────────────────────────────────────────────────────

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "md2docx-*")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	j := &job{}

	if refFiles := r.MultipartForm.File["ref"]; len(refFiles) > 0 {
		fh := refFiles[0]
		f, _ := fh.Open()
		dst := filepath.Join(tmpDir, "reference.docx")
		out, _ := os.Create(dst)
		io.Copy(out, f)
		out.Close()
		f.Close()
		j.refDoc = dst
	}

	for _, fh := range r.MultipartForm.File["files"] {
		f, err := fh.Open()
		if err != nil {
			continue
		}
		dst := filepath.Join(tmpDir, fh.Filename)
		out, err := os.Create(dst)
		if err != nil {
			f.Close()
			continue
		}
		io.Copy(out, f)
		out.Close()
		f.Close()
		j.files = append(j.files, uploadedFile{name: fh.Filename, tmpPath: dst})
	}

	if len(j.files) == 0 {
		os.RemoveAll(tmpDir)
		http.Error(w, "no valid files", http.StatusBadRequest)
		return
	}

	id := newID()
	jobs.Store(id, j)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"jobId": id})
}

// ── SSE conversion stream ─────────────────────────────────────────────────────

func handleEvents(w http.ResponseWriter, r *http.Request) {
	val, ok := jobs.Load(r.URL.Query().Get("job"))
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	j := val.(*job)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	send := func(v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", b)
		fl.Flush()
	}

	bin := findPandoc()
	if bin == "" {
		send(map[string]any{"type": "fatal", "message": "Pandoc not found. Run: " + installHint()})
		return
	}

	for _, uf := range j.files {
		base := strings.TrimSuffix(uf.name, filepath.Ext(uf.name))
		outPath := filepath.Join(filepath.Dir(uf.tmpPath), base+".docx")

		args := []string{uf.tmpPath, "-o", outPath, "--from=markdown", "--to=docx"}
		if j.refDoc != "" {
			args = append(args, "--reference-doc="+j.refDoc)
		}

		cmd := exec.Command(bin, args...)
		cmd.Stderr = os.Stderr
		err := cmd.Run()

		if err != nil {
			send(map[string]any{"type": "error", "file": uf.name, "error": err.Error()})
			continue
		}
		if _, statErr := os.Stat(outPath); statErr != nil {
			send(map[string]any{"type": "error", "file": uf.name, "error": "no output produced"})
			continue
		}

		token := newID()
		downloads.Store(token, outPath)
		send(map[string]any{
			"type":  "converted",
			"file":  uf.name,
			"docx":  base + ".docx",
			"token": token,
		})
	}

	send(map[string]any{"type": "complete"})
}

// ── download ──────────────────────────────────────────────────────────────────

func handleDownload(w http.ResponseWriter, r *http.Request) {
	val, ok := downloads.Load(r.URL.Query().Get("token"))
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	path := val.(string)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(path)+`"`)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	http.ServeFile(w, r, path)
}

// ── pandoc ────────────────────────────────────────────────────────────────────

func findPandoc() string {
	// Check alongside this binary first (app bundle)
	if exe, err := os.Executable(); err == nil {
		bundled := filepath.Join(filepath.Dir(exe), "pandoc")
		if _, err := os.Stat(bundled); err == nil {
			return bundled
		}
	}
	if p, err := exec.LookPath("pandoc"); err == nil {
		return p
	}
	for _, c := range []string{
		"/opt/homebrew/bin/pandoc",
		"/usr/local/bin/pandoc",
		`C:\Program Files\Pandoc\pandoc.exe`,
	} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
