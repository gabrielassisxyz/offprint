package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The bug this file exists to prevent: offprint needs a Chrome-family browser for PDF
// export, and without one chromedp failed with `exec: "google-chrome": executable file not
// found in $PATH` — three phases into the run, after every article had been downloaded. It
// named a binary the user might not even want (machines with `chromium` see that message),
// and offered nothing to do about it.

func TestFindBrowserAcceptsAnyChromeFamilyName(t *testing.T) {
	// chromedp looks for several names. A machine with only `chromium` is perfectly well
	// equipped, and must not be told to install "google-chrome".
	dir := t.TempDir()
	fake := filepath.Join(dir, "chromium")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	got, ok := FindBrowser()
	if !ok {
		t.Fatal("chromium on PATH was not found")
	}
	if got != fake {
		t.Fatalf("found %q, want %q", got, fake)
	}
}

func TestFindBrowserReportsAbsence(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if runtime.GOOS == "darwin" {
		t.Skip("macOS also probes /Applications, which this test cannot empty")
	}
	if _, ok := FindBrowser(); ok {
		t.Fatal("a browser was found on an empty PATH")
	}
}

func TestEnsureBrowserNamesTheFixWhenNoneIsFound(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("package-manager detection is exercised on linux")
	}
	// A PATH with a package manager but no browser: the state a fresh Linux box is in.
	dir := t.TempDir()
	pm := filepath.Join(dir, "apt-get")
	if err := os.WriteFile(pm, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	err := EnsureBrowser()
	if err == nil {
		t.Fatal("EnsureBrowser succeeded with no browser installed")
	}

	// The whole point of the change: the error must carry the command that fixes it. A
	// message the user cannot act on is the bug, not the missing browser.
	msg := err.Error()
	if !strings.Contains(msg, "apt-get install") {
		t.Errorf("error does not name the install command for this system:\n%s", msg)
	}
	if !strings.Contains(msg, "chromium") {
		t.Errorf("error does not name what to install:\n%s", msg)
	}
}

func TestInstallCommandDetectsByPackageManagerNotDistroName(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux package managers")
	}
	for _, tc := range []struct{ manager, want string }{
		{"apt-get", "apt-get"},
		{"dnf", "dnf"},
		{"pacman", "pacman"},
		{"zypper", "zypper"},
		{"apk", "apk"},
	} {
		t.Run(tc.manager, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tc.manager), []byte("#!/bin/sh\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)

			cmd := installCommand()
			if cmd == nil {
				t.Fatalf("%s on PATH produced no install command", tc.manager)
			}
			if !strings.Contains(strings.Join(cmd, " "), tc.want) {
				t.Errorf("got %v, want a command using %s", cmd, tc.want)
			}
		})
	}
}

func TestInstallCommandIsNilWhenNoPackageManagerExists(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux")
	}
	t.Setenv("PATH", t.TempDir())

	if cmd := installCommand(); cmd != nil {
		t.Fatalf("invented an install command with no package manager: %v", cmd)
	}
}

func TestIsInteractiveRejectsDevNull(t *testing.T) {
	// The subtle one. /dev/null IS a character device, so the obvious check
	// (os.ModeCharDevice) calls it interactive — and a cron job or systemd unit, whose
	// stdin is /dev/null, would print a prompt into the void, read EOF, and take the
	// silence as "no". The user gets a refusal to a question nobody ever saw.
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = devnull.Close() }()

	old := os.Stdin
	os.Stdin = devnull
	defer func() { os.Stdin = old }()

	if isInteractive() {
		t.Fatal("/dev/null was treated as an interactive terminal")
	}
}

func TestIsInteractiveRejectsAPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close(); _ = w.Close() }()

	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()

	if isInteractive() {
		t.Fatal("a pipe was treated as an interactive terminal")
	}
}
