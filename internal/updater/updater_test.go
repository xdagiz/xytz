package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallMethod(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "xytz")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		exec string
		want InstallMethod
	}{
		{name: "nix", exec: "/nix/store/abc123-xytz-0.9/bin/xytz", want: InstallNix},
		{name: "homebrew", exec: "/opt/homebrew/bin/xytz", want: InstallHomebrew},
		{name: "cellar", exec: "/opt/homebrew/Cellar/xytz/0.9/bin/xytz", want: InstallHomebrew},
		{name: "linuxbrew", exec: "/home/linuxbrew/.linuxbrew/bin/xytz", want: InstallHomebrew},
		{name: "scoop", exec: `C:\Users\me\scoop\apps\xytz\current\xytz.exe`, want: InstallScoop},
		{name: "aur", exec: "/opt/aur/xytz/bin/xytz", want: InstallAUR},
		{name: "aur /usr/bin", exec: "/usr/bin/xytz", want: InstallAUR},
		{name: "case insensitive", exec: "/NIX/STORE/xytz/bin/xytz", want: InstallNix},
		{name: "standalone", exec: binary, want: InstallStandalone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &Updater{}
			u.SetExecutablePath(tt.exec)
			if got := u.InstallMethod(); got != tt.want {
				t.Errorf("InstallMethod() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("unknown", func(t *testing.T) {
		u := &Updater{}
		if got := u.InstallMethod(); got != InstallUnknown {
			t.Errorf("InstallMethod() = %q, want %q", got, InstallUnknown)
		}
	})
}

func TestCanSelfUpdate(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "xytz")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		exec     string
		wantOK   bool
		wantHint string
	}{
		{name: "standalone binary", exec: binary, wantOK: true, wantHint: ""},
		{name: "nix install", exec: "/nix/store/abc123-xytz-0.9/bin/xytz", wantOK: false, wantHint: "nix flake"},
		{name: "homebrew install", exec: "/opt/homebrew/bin/xytz", wantOK: false, wantHint: "brew upgrade"},
		{name: "scoop install", exec: `C:\Users\me\scoop\apps\xytz\current\xytz.exe`, wantOK: false, wantHint: "scoop update"},
		{name: "aur install", exec: "/opt/aur/xytz/bin/xytz", wantOK: false, wantHint: "paru"},
		{name: "aur install /usr/bin", exec: "/usr/bin/xytz", wantOK: false, wantHint: "paru"},
		{name: "missing executable", exec: filepath.Join(t.TempDir(), "nope"), wantOK: false, wantHint: "not writable"},
		{name: "no executable", wantOK: false, wantHint: "could not locate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &Updater{}
			if tt.exec != "" {
				u.SetExecutablePath(tt.exec)
			}

			ok, hint := u.CanSelfUpdate()
			if ok != tt.wantOK {
				t.Errorf("CanSelfUpdate() ok = %v, want %v", ok, tt.wantOK)
			}

			if tt.wantHint != "" && !strings.Contains(hint, tt.wantHint) {
				t.Errorf("CanSelfUpdate() hint = %q, want substring %q", hint, tt.wantHint)
			}
		})
	}

	if runtime.GOOS != "windows" {
		t.Run("read-only binary", func(t *testing.T) {
			ro := filepath.Join(t.TempDir(), "xytz")
			if err := os.WriteFile(ro, []byte("x"), 0o444); err != nil {
				t.Fatal(err)
			}

			u := &Updater{}
			u.SetExecutablePath(ro)
			ok, hint := u.CanSelfUpdate()
			if ok {
				t.Errorf("CanSelfUpdate() ok = true, want false")
			}

			if !strings.Contains(hint, "not writable") {
				t.Errorf("CanSelfUpdate() hint = %q, want substring %q", hint, "not writable")
			}
		})

		t.Run("symlink binary", func(t *testing.T) {
			link := filepath.Join(t.TempDir(), "xytz")
			if err := os.Symlink(binary, link); err != nil {
				t.Skipf("unable to create symlink: %v", err)
			}

			u := &Updater{}
			u.SetExecutablePath(link)
			ok, hint := u.CanSelfUpdate()
			if ok {
				t.Errorf("CanSelfUpdate() ok = true, want false")
			}

			if !strings.Contains(hint, "symlink") {
				t.Errorf("CanSelfUpdate() hint = %q, want substring %q", hint, "symlink")
			}
		})
	}
}
