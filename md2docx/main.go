package main

import (
	"bufio"
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
	fmt.Println(color(bold+cyan, "  │") + color(bold+white, "   md2docx  •  v1.0.0            ") + color(bold+cyan, "│"))
	fmt.Println(color(bold+cyan, "  │") + color(dim, "   Markdown → DOCX, clean output ") + color(bold+cyan, "│"))
	fmt.Println(color(bold+cyan, "  └─────────────────────────────────┘"))
	fmt.Println()
}

// ── Pandoc detection & install ────────────────────────────────────────────────

func findPandoc() string {
	if p, err := exec.LookPath("pandoc"); err == nil {
		return p
	}
	// common non-PATH locations
	for _, c := range []string{
		"/usr/local/bin/pandoc",
		"/opt/homebrew/bin/pandoc",
		`C:\Program Files\Pandoc\pandoc.exe`,
	} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

type installPlan struct {
	tool string
	args []string
	hint string
}

func platformInstallPlan() (installPlan, bool) {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("brew"); err == nil {
			return installPlan{tool: "brew", args: []string{"install", "pandoc"}}, true
		}
		return installPlan{hint: "Install Homebrew first (https://brew.sh), then: brew install pandoc"}, false
	case "linux":
		for _, pm := range [][]string{
			{"apt-get", "sudo", "apt-get", "install", "-y", "pandoc"},
			{"dnf", "sudo", "dnf", "install", "-y", "pandoc"},
			{"pacman", "sudo", "pacman", "-S", "--noconfirm", "pandoc"},
		} {
			if _, err := exec.LookPath(pm[0]); err == nil {
				return installPlan{tool: pm[1], args: pm[2:]}, true
			}
		}
		return installPlan{hint: "Install via your package manager: pandoc"}, false
	case "windows":
		if _, err := exec.LookPath("winget"); err == nil {
			return installPlan{tool: "winget", args: []string{"install", "-e", "--id", "JohnMacFarlane.Pandoc"}}, true
		}
		return installPlan{hint: "Download from https://pandoc.org/installing.html"}, false
	}
	return installPlan{hint: "Download from https://pandoc.org/installing.html"}, false
}

func promptYN(q string) bool {
	fmt.Printf("  %s [y/N] ", q)
	s := bufio.NewScanner(os.Stdin)
	s.Scan()
	ans := strings.TrimSpace(strings.ToLower(s.Text()))
	return ans == "y" || ans == "yes"
}

func autoInstall() (string, error) {
	plan, canAuto := platformInstallPlan()

	fmt.Println(color(bold+yellow, "\n  Pandoc not found.\n"))

	if !canAuto {
		if plan.hint != "" {
			fmt.Println(color(yellow, "  "+plan.hint))
		}
		fmt.Println()
		return "", fmt.Errorf("cannot auto-install")
	}

	cmdStr := plan.tool + " " + strings.Join(plan.args, " ")
	fmt.Printf("  %s\n\n", info("Will run: "+color(bold, cmdStr)))

	if !promptYN("Install Pandoc now?") {
		fmt.Println()
		return "", fmt.Errorf("installation declined")
	}

	fmt.Println()
	sp := newSpinner("Installing Pandoc …")
	cmd := exec.Command(plan.tool, plan.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	sp.Stop()

	if err != nil {
		fmt.Println(fail("Installation failed: " + err.Error()))
		return "", err
	}
	fmt.Println(ok(color(bold, "Pandoc installed.")))
	fmt.Println()

	bin := findPandoc()
	if bin == "" {
		return "", fmt.Errorf("pandoc installed but not found in PATH — restart your shell and try again")
	}
	return bin, nil
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

// ── Progress (batch) ──────────────────────────────────────────────────────────

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

func convertFile(bin, src, outDir, refDoc string) (string, error) {
	abs, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("file not found: %s", src)
	}
	if ext := strings.ToLower(filepath.Ext(abs)); ext != ".md" && ext != ".markdown" {
		return "", fmt.Errorf("not a .md / .markdown file")
	}

	if outDir == "" {
		outDir = filepath.Dir(abs)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}

	base := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	out := filepath.Join(outDir, base+".docx")

	args := []string{abs, "-o", out, "--from=markdown", "--to=docx"}
	if refDoc != "" {
		args = append(args, "--reference-doc="+refDoc)
	}

	cmd := exec.Command(bin, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	if _, err := os.Stat(out); err != nil {
		return "", fmt.Errorf("conversion produced no output")
	}
	return out, nil
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	var (
		outDir  = flag.String("o", "", "output directory (default: same as input)")
		refDoc  = flag.String("ref", "", "reference .docx for custom Word styles/fonts")
		jobs    = flag.Int("j", 4, "parallel workers for batch conversion")
		quiet   = flag.Bool("q", false, "quiet mode: only print output paths")
		version = flag.Bool("version", false, "print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, color(bold, "\nUsage:\n"))
		fmt.Fprintf(os.Stderr, "  md2docx [flags] file.md [file2.md ...]\n")
		fmt.Fprintf(os.Stderr, "  md2docx [flags] *.md\n\n")
		fmt.Fprintf(os.Stderr, color(bold, "Flags:\n"))
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}
	flag.Parse()

	if *version {
		fmt.Println("md2docx v1.0.0")
		return
	}

	if !*quiet {
		banner()
	}

	bin := findPandoc()
	if bin == "" {
		if *quiet {
			fmt.Fprintln(os.Stderr, "error: pandoc not found. Run without -q to auto-install.")
			os.Exit(1)
		}
		var err error
		bin, err = autoInstall()
		if err != nil {
			os.Exit(1)
		}
	}

	if !*quiet {
		fmt.Println(info("Backend: " + color(bold, "Pandoc")))
		fmt.Println()
	}

	files := flag.Args()
	if len(files) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// single file
	if len(files) == 1 {
		var sp *spinner
		if !*quiet {
			sp = newSpinner("Converting " + color(bold, files[0]) + " …")
		}
		out, err := convertFile(bin, files[0], *outDir, *refDoc)
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

	// batch
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
			out, err := convertFile(bin, f, *outDir, *refDoc)
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
