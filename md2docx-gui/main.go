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
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	mux := http.NewServeMux()
	webFS, _ := fs.Sub(webFiles, "web")
	mux.Handle("/", http.FileServer(http.FS(webFS)))
	mux.HandleFunc("/upload-by-path", handleUploadByPath)
	mux.HandleFunc("/events", handleEvents)
	mux.HandleFunc("/check", handleCheck)

	server = &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}
	go server.ListenAndServe()

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("md2docx")
	w.SetSize(660, 780, webview.HintNone)

	// Native file picker — WKWebView blocks hidden <input type="file"> clicks
	w.Bind("pickFiles", func() []string {
		out, err := exec.Command("osascript",
			"-e", `set theFiles to choose file of type {"public.data"} with multiple selections allowed with prompt "Select Markdown files to convert"`,
			"-e", `set output to ""`,
			"-e", `repeat with f in theFiles`,
			"-e", `set output to output & POSIX path of f & linefeed`,
			"-e", `end repeat`,
			"-e", `return output`,
		).Output()
		if err != nil {
			return nil // user cancelled
		}
		var paths []string
		for _, p := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			ext := strings.ToLower(filepath.Ext(p))
			if ext == ".md" || ext == ".markdown" {
				paths = append(paths, p)
			}
		}
		return paths
	})

	// Native reference .docx picker
	w.Bind("pickRefDoc", func() string {
		out, err := exec.Command("osascript",
			"-e", `set theFile to choose file of type {"public.data"} with prompt "Select a reference .docx to apply its styles"`,
			"-e", `return POSIX path of theFile`,
		).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	})

	// Native Save As dialog — WKWebView ignores <a download> links
	w.Bind("saveFile", func(token string, suggestedName string) bool {
		val, ok := downloads.Load(token)
		if !ok {
			return false
		}
		srcPath := val.(string)

		out, err := exec.Command("osascript",
			"-e", fmt.Sprintf(`set savePath to choose file name default name "%s" with prompt "Save converted file:"`, suggestedName),
			"-e", `return POSIX path of savePath`,
		).Output()
		if err != nil {
			return false // user cancelled
		}
		dstPath := strings.TrimSpace(string(out))
		if !strings.HasSuffix(strings.ToLower(dstPath), ".docx") {
			dstPath += ".docx"
		}

		src, err := os.Open(srcPath)
		if err != nil {
			return false
		}
		defer src.Close()
		dst, err := os.Create(dstPath)
		if err != nil {
			return false
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		return err == nil
	})

	w.Navigate(url)
	w.Run() // blocks until window closed

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

// ── upload by filesystem path (used by native file picker) ───────────────────

func handleUploadByPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Paths  []string `json:"paths"`
		RefDoc string   `json:"refDoc"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	j := &job{refDoc: req.RefDoc}
	for _, path := range req.Paths {
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		j.files = append(j.files, uploadedFile{
			name:    filepath.Base(path),
			tmpPath: path,
		})
	}

	if len(j.files) == 0 {
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

		// Always write output to a temp file so we control the path
		tmpOut, err := os.CreateTemp("", base+"-*.docx")
		if err != nil {
			send(map[string]any{"type": "error", "file": uf.name, "error": "could not create temp file"})
			continue
		}
		outPath := tmpOut.Name()
		tmpOut.Close()

		args := []string{uf.tmpPath, "-o", outPath, "--from=markdown", "--to=docx"}
		if j.refDoc != "" {
			args = append(args, "--reference-doc="+j.refDoc)
		}

		cmd := exec.Command(bin, args...)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Remove(outPath)
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

// ── pandoc ────────────────────────────────────────────────────────────────────

func findPandoc() string {
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
	} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
