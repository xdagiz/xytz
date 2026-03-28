package paths

import (
	"os"
	"runtime"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		dir := GetConfigDir()
		if dir == "" {
			t.Error("GetConfigDir() returned empty string")
		}
	})

	t.Run("ends with xytz", func(t *testing.T) {
		dir := GetConfigDir()
		expectedSuffix := "xytz"
		if len(dir) < len(expectedSuffix) || dir[len(dir)-len(expectedSuffix):] != expectedSuffix {
			t.Errorf("GetConfigDir() = %q, should end with %q", dir, expectedSuffix)
		}
	})
}

func TestGetDataDir(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		dir := GetDataDir()
		if dir == "" {
			t.Error("GetDataDir() returned empty string")
		}
	})

	t.Run("ends with xytz", func(t *testing.T) {
		dir := GetDataDir()
		expectedSuffix := "xytz"
		if len(dir) < len(expectedSuffix) || dir[len(dir)-len(expectedSuffix):] != expectedSuffix {
			t.Errorf("GetDataDir() = %q, should end with %q", dir, expectedSuffix)
		}
	})
}

func TestEnsureDirExists(t *testing.T) {
	t.Run("creates directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		testDir := tmpDir + "/newdir"
		err = EnsureDirExists(testDir)
		if err != nil {
			t.Errorf("EnsureDirExists() error = %v", err)
		}

		// Verify directory was created
		info, err := os.Stat(testDir)
		if err != nil {
			t.Errorf("Failed to stat directory: %v", err)
		}
		if !info.IsDir() {
			t.Error("Expected path to be a directory")
		}
	})

	t.Run("succeeds if directory exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		err = EnsureDirExists(tmpDir)
		if err != nil {
			t.Errorf("EnsureDirExists() error = %v", err)
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "xytz-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		nestedDir := tmpDir + "/level1/level2/level3"
		err = EnsureDirExists(nestedDir)
		if err != nil {
			t.Errorf("EnsureDirExists() error = %v", err)
		}

		info, err := os.Stat(nestedDir)
		if err != nil {
			t.Errorf("Failed to stat directory: %v", err)
		}
		if !info.IsDir() {
			t.Error("Expected path to be a directory")
		}
	})
}

func TestPlatformSpecificPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific tests on Windows")
	}

	if runtime.GOOS == "linux" {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		dir := GetConfigDir()
		if dir != "/custom/config/xytz" {
			t.Errorf("GetConfigDir() = %q, expected XDG_CONFIG_HOME to be respected", dir)
		}

		t.Setenv("XDG_DATA_HOME", "/custom/data")
		dir = GetDataDir()
		if dir != "/custom/data/xytz" {
			t.Errorf("GetDataDir() = %q, expected XDG_DATA_HOME to be respected", dir)
		}
	}

	if runtime.GOOS == "darwin" {
		t.Run("XDG_CONFIG_HOME respected on darwin", func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", "/custom/config")
			dir := GetConfigDir()
			if dir != "/custom/config/xytz" {
				t.Errorf("GetConfigDir() = %q, expected XDG_CONFIG_HOME to be respected on darwin", dir)
			}
		})

		t.Run("XDG_DATA_HOME respected on darwin", func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", "/custom/data")
			dir := GetDataDir()
			if dir != "/custom/data/xytz" {
				t.Errorf("GetDataDir() = %q, expected XDG_DATA_HOME to be respected on darwin", dir)
			}
		})
	}
}
