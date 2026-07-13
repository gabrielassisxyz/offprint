package app

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// PDF export drives a headless Chrome through chromedp, which shells out to a browser
// binary it expects to find on the system. Without one, chromedp fails with
// `exec: "google-chrome": executable file not found in $PATH` — a message that names a
// binary the user may not even be meant to install (this machine has `chromium`), says
// nothing about what to do, and arrives three phases deep, after every article has already
// been downloaded and rendered.
//
// So the check runs up front, and when it fails it offers to fix itself.

// candidates are the names chromedp itself looks for, in its order of preference.
var candidates = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"chrome",
	"headless_shell",
}

// FindBrowser returns the path of the first Chrome-family browser on the system.
func FindBrowser() (string, bool) {
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, true
		}
	}
	// macOS ships Chrome inside a bundle, which is not on anyone's PATH.
	if runtime.GOOS == "darwin" {
		for _, p := range []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		} {
			if _, err := os.Stat(p); err == nil {
				return p, true
			}
		}
	}
	return "", false
}

// installCommand returns the package-manager command that would install Chromium here,
// or nil when we cannot tell. Detection is by package manager rather than by distro
// name: what matters is how software gets installed, not what the OS calls itself.
func installCommand() []string {
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("brew"); err == nil {
			return []string{"brew", "install", "--cask", "chromium"}
		}
		return nil
	}
	if runtime.GOOS != "linux" {
		return nil // Windows: no package manager we can rely on.
	}
	switch {
	case has("apt-get"):
		return []string{"sudo", "apt-get", "install", "-y", "chromium"}
	case has("dnf"):
		return []string{"sudo", "dnf", "install", "-y", "chromium"}
	case has("pacman"):
		return []string{"sudo", "pacman", "-S", "--noconfirm", "chromium"}
	case has("zypper"):
		return []string{"sudo", "zypper", "install", "-y", "chromium"}
	case has("apk"):
		return []string{"sudo", "apk", "add", "chromium"}
	}
	return nil
}

func has(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// EnsureBrowser makes sure a browser exists before any work begins, offering to install
// one if not. It returns an error rather than exiting, so the caller decides.
//
// The install is *offered*, never silent: it needs sudo, and a tool that quietly runs a
// package manager as root because you asked it to make a PDF has overstepped. When stdin
// is not a terminal — CI, a pipe, a cron job — there is nobody to ask, so it explains and
// fails instead of hanging on a prompt nobody will ever see.
func EnsureBrowser() error {
	if _, ok := FindBrowser(); ok {
		return nil // The common case: say nothing, get on with it.
	}

	color.Yellow("PDF export needs a Chrome or Chromium browser, and none was found.")
	fmt.Println("(`--format html` skips PDF and needs no browser.)")
	fmt.Println()

	cmd := installCommand()
	if cmd == nil {
		return fmt.Errorf("no browser found. Install Chrome or Chromium:\n%s", manualHint())
	}

	if !isInteractive() {
		return fmt.Errorf("no browser found. Install one with:\n\n    %s\n\n%s",
			strings.Join(cmd, " "), manualHint())
	}

	fmt.Printf("Install it now?  %s\n", color.CyanString(strings.Join(cmd, " ")))
	fmt.Print("[y/N] ")

	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		return fmt.Errorf("no browser. Install one and try again:\n\n    %s\n\n%s",
			strings.Join(cmd, " "), manualHint())
	}

	fmt.Println()
	run := exec.Command(cmd[0], cmd[1:]...)
	run.Stdout, run.Stderr, run.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := run.Run(); err != nil {
		// A failed install is not a dead end — it just means the user has to do it
		// themselves, and they should not have to go hunting for how.
		return fmt.Errorf("install failed: %w\n\nInstall Chrome or Chromium by hand:\n%s",
			err, manualHint())
	}

	// Trust nothing: a package manager exiting 0 is not the same as a browser existing.
	if _, ok := FindBrowser(); !ok {
		return fmt.Errorf("the install reported success but no browser was found on PATH.\n%s",
			manualHint())
	}

	color.Green("Browser installed.")
	fmt.Println()
	return nil
}

// isInteractive reports whether there is a human on the other end of stdin.
//
// The obvious implementation — checking os.ModeCharDevice — is wrong, and wrong in the
// case that matters: **/dev/null is a character device**. A service run from cron or
// systemd, whose stdin is /dev/null, would sail past that check, print a prompt into the
// void, read EOF, and treat the silence as "no". The user gets a mysterious refusal from a
// question nobody was ever shown.
//
// A pipe is correctly caught either way. It is the /dev/null case that bites.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func manualHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "  https://www.google.com/chrome/  (or: brew install --cask chromium)"
	case "windows":
		return "  https://www.google.com/chrome/"
	default:
		return "  https://www.google.com/chrome/  — or your distribution's `chromium` package"
	}
}
