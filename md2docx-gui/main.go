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

	shutdownMu    sync.Mutex
	shutdownTimer *time.Timer
	server        *http.Server
)

const heartbeatTimeout = 12 * time.Second

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
	mux.HandleFunc("/upload", handleUpload)
	mux.HandleFunc("/events", handleEvents)
	mux.HandleFunc("/download", handleDownload)
	mux.HandleFunc("/heartbeat", handleHeartbeat)
	mux.HandleFunc("/check", handleCheck)

	server = &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Println("md2docx →", url)
	fmt.Println("Close the browser tab to quit.")

	resetShutdownTimer()
	go openBrowser(url)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resetShutdownTimer() {
	shutdownMu.Lock()
	defer shutdownMu.Unlock()
	if shutdownTimer != nil {
		shutdownTimer.Reset(heartbeatTimeout)
		return
	}
	shutdownTimer = time.AfterFunc(heartbeatTimeout, func() {
		fmt.Println("Browser tab closed. Shutting down.")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	})
}

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	resetShutdownTimer()
	w.WriteHeader(http.StatusOK)
}

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
		send(map[string]any{"type": "fatal", "message": "Pandoc not found. " + installHint()})
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
		send(map[string]any{"type": "converted", "file": uf.name, "docx": base + ".docx", "token": token})
	}

	send(map[string]any{"type": "complete"})
}

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
	for _, c := range []string{"/opt/homebrew/bin/pandoc", "/usr/local/bin/pandoc"} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func openBrowser(url string) {
	time.Sleep(350 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Run()
	}
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
