package gh

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// localBinDir is where we install gh when it isn't already on PATH.
// Mirrors git-switch's own install location.
const localBinDir = ".local/bin"

// ghBinary returns the path used to invoke gh. Prefers a system install on
// PATH; falls back to the user-local copy installed by Install().
func ghBinary() string {
	if path, err := exec.LookPath("gh"); err == nil {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "gh" // best-effort; will fail loudly downstream
	}
	return filepath.Join(home, localBinDir, "gh")
}

// IsInstalled reports whether gh is available, either on PATH or in the
// user-local bin directory.
func IsInstalled() bool {
	if _, err := exec.LookPath("gh"); err == nil {
		return true
	}
	if _, err := os.Stat(ghBinary()); err == nil {
		return true
	}
	return false
}

// EnsureInstalled installs gh into ~/.local/bin if it isn't already available.
func EnsureInstalled() error {
	if IsInstalled() {
		return nil
	}
	fmt.Println("==> GitHub CLI (gh) not found. Installing to ~/.local/bin/gh ...")
	if err := Install(); err != nil {
		return fmt.Errorf("installing gh: %w", err)
	}
	fmt.Println("==> gh installed.")
	return nil
}

// Install downloads the latest gh release for the current OS/arch and places
// the binary in ~/.local/bin/gh. Uses no sudo.
func Install() error {
	osName, archName, ext, err := platformAsset()
	if err != nil {
		return err
	}

	tag, err := latestGhTag()
	if err != nil {
		return err
	}
	version := strings.TrimPrefix(tag, "v")

	asset := fmt.Sprintf("gh_%s_%s_%s.%s", version, osName, archName, ext)
	url := fmt.Sprintf("https://github.com/cli/cli/releases/download/%s/%s", tag, asset)

	tmpDir, err := os.MkdirTemp("", "gh-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, asset)
	if err := download(url, archivePath); err != nil {
		return err
	}

	innerDir := fmt.Sprintf("gh_%s_%s_%s", version, osName, archName)
	if err := extractGh(archivePath, ext, innerDir, tmpDir); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dstDir := filepath.Join(home, localBinDir)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(dstDir, "gh")

	src := filepath.Join(tmpDir, "gh")
	if err := moveFile(src, dst); err != nil {
		return err
	}
	return os.Chmod(dst, 0o755)
}

// platformAsset maps GOOS/GOARCH to the (os, arch, ext) tuple used by
// gh's release filenames.
func platformAsset() (string, string, string, error) {
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		return "", "", "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
	switch runtime.GOOS {
	case "darwin":
		return "macOS", arch, "zip", nil
	case "linux":
		return "linux", arch, "tar.gz", nil
	default:
		return "", "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// latestGhTag returns the tag name of cli/cli's latest release, e.g. "v2.50.0".
func latestGhTag() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/cli/cli/releases/latest")
	if err != nil {
		return "", fmt.Errorf("fetching latest gh release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned %s", resp.Status)
	}
	var meta struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", fmt.Errorf("decoding release metadata: %w", err)
	}
	if meta.TagName == "" {
		return "", fmt.Errorf("github returned empty tag_name")
	}
	return meta.TagName, nil
}

// download fetches url to dst.
func download(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing %s: %w", dst, err)
	}
	return nil
}

// extractGh pulls the gh binary out of the release archive and writes it to
// tmpDir/gh. innerDir is the top-level directory the archive contains, e.g.
// "gh_2.50.0_macOS_arm64".
func extractGh(archivePath, ext, innerDir, tmpDir string) error {
	wantPath := innerDir + "/bin/gh"
	dst := filepath.Join(tmpDir, "gh")

	switch ext {
	case "zip":
		return extractFromZip(archivePath, wantPath, dst)
	case "tar.gz":
		return extractFromTarGz(archivePath, wantPath, dst)
	default:
		return fmt.Errorf("unknown archive format: %s", ext)
	}
}

func extractFromZip(archivePath, wantPath, dst string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != wantPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return writeFromReader(rc, dst)
	}
	return fmt.Errorf("gh binary not found in archive at %s", wantPath)
}

func extractFromTarGz(archivePath, wantPath, dst string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("opening gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}
		if hdr.Name != wantPath {
			continue
		}
		return writeFromReader(tr, dst)
	}
	return fmt.Errorf("gh binary not found in archive at %s", wantPath)
}

func writeFromReader(r io.Reader, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, r)
	return err
}

// moveFile renames if possible; falls back to copy+delete across filesystems.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}
