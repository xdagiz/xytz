package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	log "charm.land/log/v2"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/xdagiz/xytz/internal/version"
)

const ChecksumFileName = "checksums.txt"

var (
	ErrNotConfigured = errors.New("updater not configured")
	ErrMissingSource = errors.New("invalid release: missing source data")
	ErrNoExecutable  = errors.New("could not locate executable path")
)

type InstallMethod string

const (
	InstallStandalone InstallMethod = "standalone"
	InstallHomebrew   InstallMethod = "homebrew"
	InstallNix        InstallMethod = "nix"
	InstallScoop      InstallMethod = "scoop"
	InstallAUR        InstallMethod = "aur"
	InstallUnknown    InstallMethod = "unknown"
)

type Release struct {
	Version      string
	ReleaseNotes string
	AssetName    string
	AssetSize    int
	src          *selfupdate.Release
}

type Updater struct {
	updater  *selfupdate.Updater
	repo     selfupdate.Repository
	execPath string
}

type UpdateService interface {
	DetectLatest(ctx context.Context) (*Release, bool, error)
	Install(ctx context.Context, rel *Release) error
	CanSelfUpdate() (ok bool, hint string)
}

func New() *Updater {
	up, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: ChecksumFileName},
	})
	if err != nil {
		log.Warn("failed to initialize updater", "err", err)
	}

	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}

	return &Updater{
		updater:  up,
		repo:     selfupdate.ParseSlug(version.RepoSlug),
		execPath: exe,
	}
}

func NewWithPath(execPath string) *Updater {
	up := New()
	up.execPath = execPath
	return up
}

func (u *Updater) SetExecutablePath(path string) {
	u.execPath = path
}

func (u *Updater) DetectLatest(ctx context.Context) (*Release, bool, error) {
	if u.updater == nil {
		return nil, false, ErrNotConfigured
	}

	rel, found, err := u.updater.DetectLatest(ctx, u.repo)
	if err != nil || !found {
		return nil, found, err
	}

	return &Release{
		Version:      rel.Version(),
		ReleaseNotes: rel.ReleaseNotes,
		AssetName:    rel.AssetName,
		AssetSize:    rel.AssetByteSize,
		src:          rel,
	}, true, nil
}

func (u *Updater) Install(ctx context.Context, rel *Release) error {
	src, err := u.releaseSource(rel)
	if err != nil {
		return err
	}

	cmdPath, err := u.executablePath()
	if err != nil {
		return err
	}

	return u.updater.UpdateTo(ctx, src, cmdPath)
}

func (u *Updater) InstallTo(ctx context.Context, rel *Release, target string) error {
	src, err := u.releaseSource(rel)
	if err != nil {
		return err
	}
	return u.updater.UpdateTo(ctx, src, target)
}

func (u *Updater) releaseSource(rel *Release) (*selfupdate.Release, error) {
	if u.updater == nil {
		return nil, ErrNotConfigured
	}

	if rel == nil || rel.src == nil {
		return nil, ErrMissingSource
	}

	return rel.src, nil
}

func (u *Updater) executablePath() (string, error) {
	if u.execPath != "" {
		resolved, err := filepath.EvalSymlinks(u.execPath)
		if err == nil {
			return resolved, nil
		}

		return u.execPath, nil
	}

	return "", ErrNoExecutable
}

func (u *Updater) InstallMethod() InstallMethod {
	path, err := u.executablePath()
	if err != nil {
		return InstallUnknown
	}

	lower := strings.ToLower(path)

	switch {
	case strings.Contains(lower, "/homebrew/"),
		strings.Contains(lower, "/cellar/"),
		strings.Contains(lower, "/linuxbrew/"):
		return InstallHomebrew

	case strings.Contains(lower, "/nix/store/"):
		return InstallNix

	case strings.Contains(lower, "scoop/apps/"), strings.Contains(lower, "scoop\\apps\\"):
		return InstallScoop

	case strings.Contains(lower, "/aur/"),
		strings.HasPrefix(lower, "/usr/bin/"):
		return InstallAUR
	}

	return InstallStandalone
}

func (u *Updater) CanSelfUpdate() (ok bool, hint string) {
	switch u.InstallMethod() {
	case InstallHomebrew:
		return false, "xytz was installed with Homebrew - update with: brew upgrade xytz"
	case InstallNix:
		return false, "xytz was installed with Nix - update your flake input (e.g. nix flake update)"
	case InstallScoop:
		return false, "xytz was installed with Scoop - update with: scoop update xytz"
	case InstallAUR:
		return false, "xytz was installed from the AUR - update with: paru -Syu (or yay -Syu)"
	}

	if u.execPath != "" {
		if info, err := os.Lstat(u.execPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return false, "xytz is a symlink - reinstall with the installer script: curl -fsSL https://raw.githubusercontent.com/xdagiz/xytz/main/install.sh | bash"
		}
	}

	path, err := u.executablePath()
	if err != nil {
		return false, "could not locate the xytz executable"
	}

	if !isWritable(path) {
		return false, "the xytz binary is not writable - reinstall with the installer script (curl -fsSL https://raw.githubusercontent.com/xdagiz/xytz/main/install.sh | bash) or your package manager"
	}

	return true, ""
}

func isWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if runtime.GOOS != "windows" && info.Mode().Perm()&0o200 == 0 {
		return false
	}

	if runtime.GOOS == "windows" {
		return true
	}

	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".xytz-write-test-*")
	if err != nil {
		return false
	}

	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)

	return true
}

func VersionDisplay(v string) string {
	v = version.NormalizeVersion(v)
	if v == "" {
		return v
	}

	return "v" + v
}

func (u *Updater) CurrentExecutable() string {
	path, err := u.executablePath()
	if err != nil {
		return ""
	}

	return path
}

func (m InstallMethod) String() string {
	return string(m)
}
