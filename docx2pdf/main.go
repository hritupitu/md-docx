package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ── ANSI colours ──────────────────────────────────────────────────────────────

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	white  = "\033[97m"
)

func color(c, s string) string { return c + s + reset }
func ok(s string) string       { return color(green, "✔ ") + s }
func fail(s string) string     { return color(red, "✘ ") + s }
func info(s string) string     { return color(cyan, "→ ") + s }

// ── Banner ────────────────────────────────────────────────────────────────────

func banner() {
	fmt.Println()
	fmt.Println(color(bold+cyan, "  ┌─────────────────────────────────┐"))
	fmt.Println(color(bold+cyan, "  │") + color(bold+white, "   docx2pdf  •  v1.0.0           ") + color(bold+cyan, "│"))
	fmt.Println(color(bold+cyan, "  │") + color(dim, "   DOCX → PDF, formatting intact ") + color(bold+cyan, "│"))
	fmt.Println(color(bold+cyan, "  └─────────────────────────────────┘"))
	fmt.Println()
}

// ── Backend detection ─────────────────────────────────────────────────────────

type backend struct {
	name    string
	convert func(src, outDir string) error
}

func findSoffice() string {
	candidates := []string{
		"soffice",
		"libreoffice",
		"/Applications/LibreOffice.app/Contents/MacOS/soffice",
		"/usr/lib/libreoffice/program/soffice",
		`C:\Program Files\LibreOffice\program\soffice.exe`,
	}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func libreofficeBackend(bin string) backend {
	return backend{
		name: "LibreOffice",
		convert: func(src, outDir string) error {
			cmd := exec.Command(bin,
				"--headless",
				"--norestore",
				"--nofirststartwizard",
				"--convert-to", "pdf:writer_pdf_Export",
				"--outdir", outDir,
				src,
			)
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}
}

func unoconvBackend() (backend, bool) {
	p, err := exec.LookPath("unoconv")
	if err != nil {
		return backend{}, false
	}
	return backend{
		name: "unoconv",
		convert: func(src, outDir string) error {
			base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
			out := filepath.Join(outDir, base+".pdf")
			return exec.Command(p, "-f", "pdf", "-o", out, src).Run()
		},
	}, true
}

func detectBackend() (backend, error) {
	if bin := findSoffice(); bin != "" {
		return libreofficeBackend(bin), nil
	}
	if b, ok := unoconvBackend(); ok {
		return b, nil
	}
	return backend{}, errors.New("no converter found")
}

// ── Auto-install ──────────────────────────────────────────────────────────────

type installPlan struct {
	tool string
	args []string
	hint string // shown if the tool itself isn't available
}

func platformInstallPlan() (installPlan, bool) {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("brew"); err == nil {
			return installPlan{
				tool: "brew",
				args: []string{"install", "--cask", "libreoffice"},
			}, true
		}
		return installPlan{hint: "Install Homebrew first: https://brew.sh"}, false

	case "linux":
		if _, err := exec.LookPath("apt-get"); err == nil {
			return installPlan{
				tool: "sudo",
				args: []string{"apt-get", "install", "-y", "libreoffice"},
			}, true
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			return installPlan{
				tool: "sudo",
				args: []string{"dnf", "install", "-y", "libreoffice"},
			}, true
		}
		if _, err := exec.LookPath("pacman"); err == nil {
			return installPlan{
				tool: "sudo",
				args: []string{"pacman", "-S", "--noconfirm", "libreoffice-still"},
			}, true
		}
		return installPlan{hint: "Install via your package manager: libreoffice"}, false

	case "windows":
		if _, err := exec.LookPath("winget"); err == nil {
			return installPlan{
				tool: "winget",
				args: []string{"install", "-e", "--id", "TheDocumentFoundation.LibreOffice"},
			}, true
		}
		return installPlan{hint: "Download from https://www.libreoffice.org/download"}, false
	}
	return installPlan{hint: "Download from https://www.libreoffice.org/download"}, false
}

func promptYN(question string) bool {
	fmt.Printf("  %s [y/N] ", question)
	s := bufio.NewScanner(os.Stdin)
	s.Scan()
	ans := strings.TrimSpace(strings.ToLower(s.Text()))
	return ans == "y" || ans == "yes"
}

// autoInstall tries to install LibreOffice and returns the detected backend.
func autoInstall() (backend, error) {
	plan, canAuto := platformInstallPlan()

	if !canAuto {
		fmt.Println(color(bold+red, "\n  LibreOffice not found.\n"))
		if plan.hint != "" {
			fmt.Println(color(yellow, "  "+plan.hint))
		}
		fmt.Println()
		return backend{}, errors.New("cannot auto-install")
	}

	cmdStr := plan.tool + " " + strings.Join(plan.args, " ")
	fmt.Println(color(bold+yellow, "\n  LibreOffice not found.\n"))
	fmt.Printf("  %s\n\n", info("Will run: "+color(bold, cmdStr)))

	if !promptYN("Install LibreOffice now?") {
		fmt.Println()
		return backend{}, errors.New("installation declined")
	}

	fmt.Println()
	sp := newSpinner("Installing LibreOffice (this may take a few minutes) …")
	cmd := exec.Command(plan.tool, plan.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	sp.Stop()

	if err != nil {
		fmt.Println(fail("Installation failed: " + err.Error()))
		return backend{}, err
	}
	fmt.Println(ok(color(bold, "LibreOffice installed.")))
	fmt.Println()

	b, err := detectBackend()
	if err != nil {
		return backend{}, errors.New("LibreOffice installed but still not found in PATH — restart your shell and try again")
	}
	return b, nil
}

// ── Spinner ───────────────────────────────────────────────────────────────────

type spinner struct {
	stop chan struct{}
	done chan struct{}
}

func newSpinner(msg string) *spinner {
	s := &spinner{stop: make(chan struct{}), done: make(chan struct{})}
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Printf("\r\033[K")
				return
			default:
				fmt.Printf("\r  %s %s", color(cyan, frames[i%len(frames)]), msg)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()
	return s
}

func (s *spinner) Stop() { close(s.stop); <-s.done }

// ── Progress bar (batch) ──────────────────────────────────────────────────────

type progress struct {
	total  int32
	done   atomic.Int32
	failed atomic.Int32
	mu     sync.Mutex
}

func (p *progress) tick(file string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	d := p.done.Add(1)
	if err != nil {
		p.failed.Add(1)
		fmt.Printf("\r\033[K%s\n", fail(filepath.Base(file)+": "+err.Error()))
	} else {
		fmt.Printf("\r\033[K%s\n", ok(filepath.Base(file)))
	}
	pct := int(float64(d) / float64(p.total) * 30)
	bar := strings.Repeat("█", pct) + strings.Repeat("░", 30-pct)
	fmt.Printf("  [%s] %d/%d\r", color(cyan, bar), d, p.total)
}

// ── Conversion ────────────────────────────────────────────────────────────────

func convertFile(b backend, src, outDir string) (string, error) {
	abs, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("file not found: %s", src)
	}
	if ext := strings.ToLower(filepath.Ext(abs)); ext != ".docx" && ext != ".doc" {
		return "", fmt.Errorf("not a .docx / .doc file")
	}
	if outDir == "" {
		outDir = filepath.Dir(abs)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	if err := b.convert(abs, outDir); err != nil {
		return "", err
	}
	base := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	out := filepath.Join(outDir, base+".pdf")
	if _, err := os.Stat(out); err != nil {
		return "", fmt.Errorf("conversion produced no output")
	}
	return out, nil
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	var (
		outDir  = flag.String("o", "", "output directory (default: same as input file)")
		jobs    = flag.Int("j", 4, "parallel workers for batch conversion")
		quiet   = flag.Bool("q", false, "quiet mode: only print output paths, no prompts")
		version = flag.Bool("version", false, "print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, color(bold, "\nUsage:\n"))
		fmt.Fprintf(os.Stderr, "  docx2pdf [flags] file.docx [file2.docx ...]\n")
		fmt.Fprintf(os.Stderr, "  docx2pdf [flags] *.docx\n\n")
		fmt.Fprintf(os.Stderr, color(bold, "Flags:\n"))
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}
	flag.Parse()

	if *version {
		fmt.Println("docx2pdf v1.0.0")
		return
	}

	if !*quiet {
		banner()
	}

	// ── detect or auto-install backend ──
	b, err := detectBackend()
	if err != nil {
		if *quiet {
			fmt.Fprintln(os.Stderr, "error: LibreOffice not found. Run without -q to auto-install.")
			os.Exit(1)
		}
		b, err = autoInstall()
		if err != nil {
			os.Exit(1)
		}
	}

	if !*quiet {
		fmt.Println(info("Backend: " + color(bold, b.name)))
		fmt.Println()
	}

	files := flag.Args()
	if len(files) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// ── single file ──
	if len(files) == 1 {
		var sp *spinner
		if !*quiet {
			sp = newSpinner("Converting " + color(bold, files[0]) + " …")
		}
		out, err := convertFile(b, files[0], *outDir)
		if sp != nil {
			sp.Stop()
		}
		if err != nil {
			fmt.Println(fail("Failed: " + err.Error()))
			os.Exit(1)
		}
		if *quiet {
			fmt.Println(out)
		} else {
			fmt.Println(ok(color(bold, out)))
			fmt.Println()
		}
		return
	}

	// ── batch ──
	if !*quiet {
		fmt.Printf("  %s Converting %s%d files%s …\n\n",
			color(cyan, "⚡"), bold, len(files), reset)
	}

	pr := &progress{total: int32(len(files))}
	sem := make(chan struct{}, *jobs)
	var wg sync.WaitGroup

	for _, f := range files {
		f := f
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			out, err := convertFile(b, f, *outDir)
			if *quiet {
				if err == nil {
					fmt.Println(out)
				}
			} else {
				pr.tick(f, err)
			}
		}()
	}
	wg.Wait()

	if !*quiet {
		fmt.Printf("\r\033[K\n")
		total := int(pr.total)
		done := int(pr.done.Load())
		failed := int(pr.failed.Load())
		if failed == 0 {
			fmt.Printf("  %s %d/%d files converted successfully\n\n",
				color(green+bold, "✔"), done, total)
		} else {
			fmt.Printf("  %s %d/%d converted, %s failed\n\n",
				color(yellow+bold, "⚠"), done-failed, total,
				color(red, fmt.Sprintf("%d", failed)))
		}
	}
	if pr.failed.Load() > 0 {
		os.Exit(1)
	}
}
