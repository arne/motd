package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	releaseAPI    = "https://api.github.com/repos/arne/motd/releases/latest"
	httpUserAgent = "motd-self-update"
)

type release struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func runUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	check := fs.Bool("check", false, "report the latest version without installing")
	force := fs.Bool("force", false, "reinstall even if the latest version matches the current one")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rel, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("fetch latest release: %w", err)
	}

	current := versionTag(version)

	if *check {
		fmt.Printf("current: %s\nlatest:  %s\n", current, rel.TagName)
		return nil
	}

	if current == rel.TagName && !*force {
		fmt.Printf("already on latest (%s)\n", rel.TagName)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate motd binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	dir := filepath.Dir(exe)
	if err := checkWritable(dir); err != nil {
		return fmt.Errorf("%s is not writable — re-run with sudo, or reinstall manually (see README)", dir)
	}

	tarballName, err := assetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	var tarballURL, checksumsURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case tarballName:
			tarballURL = a.URL
		case "checksums.txt":
			checksumsURL = a.URL
		}
	}
	if tarballURL == "" {
		return fmt.Errorf("release %s has no asset named %s", rel.TagName, tarballName)
	}
	if checksumsURL == "" {
		return fmt.Errorf("release %s has no checksums.txt", rel.TagName)
	}

	fmt.Printf("downloading %s ...\n", rel.TagName)

	tarballBytes, err := downloadAll(tarballURL)
	if err != nil {
		return fmt.Errorf("download tarball: %w", err)
	}

	checksums, err := downloadAll(checksumsURL)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}

	if err := verifyChecksum(checksums, tarballName, tarballBytes); err != nil {
		return fmt.Errorf("verify %s: %w", tarballName, err)
	}

	if err := installBinary(exe, tarballBytes); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	fmt.Printf("upgraded %s -> %s\n", current, rel.TagName)
	return nil
}

// versionTag normalizes the ldflags-injected version string into a
// GitHub-release-style tag (e.g. "0.1.2" -> "v0.1.2"). Empty or "dev"
// builds round-trip as "dev".
func versionTag(v string) string {
	if v == "" || v == "dev" {
		return "dev"
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

func fetchLatestRelease() (*release, error) {
	req, err := http.NewRequest("GET", releaseAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", httpUserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return nil, fmt.Errorf("GitHub API %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, errors.New("empty tag_name from GitHub API")
	}
	return &rel, nil
}

// assetName returns the goreleaser archive name for the given GOOS/GOARCH.
// Mirrors the name_template in .goreleaser.yml.
func assetName(goos, goarch string) (string, error) {
	var osPart string
	switch goos {
	case "linux":
		osPart = "Linux"
	case "darwin":
		osPart = "Darwin"
	default:
		return "", fmt.Errorf("unsupported OS: %s", goos)
	}
	var archPart string
	switch goarch {
	case "amd64":
		archPart = "x86_64"
	case "arm64":
		archPart = "arm64"
	default:
		return "", fmt.Errorf("unsupported arch: %s", goarch)
	}
	return fmt.Sprintf("motd_%s_%s.tar.gz", osPart, archPart), nil
}

func downloadAll(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", httpUserAgent)
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s for %s", resp.Status, url)
	}
	return io.ReadAll(resp.Body)
}

// verifyChecksum reads a goreleaser checksums.txt and confirms the SHA256
// of data matches the entry for name.
func verifyChecksum(checksums []byte, name string, data []byte) error {
	var want string
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == name {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("no checksum for %s", name)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", want, got)
	}
	return nil
}

func installBinary(exe string, tarball []byte) error {
	gz, err := gzip.NewReader(bytes.NewReader(tarball))
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return errors.New("no motd binary in tarball")
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg || path.Clean(h.Name) != "motd" {
			continue
		}
		dir := filepath.Dir(exe)
		tmp, err := os.CreateTemp(dir, ".motd-update-*")
		if err != nil {
			return err
		}
		tmpName := tmp.Name()
		_, copyErr := io.Copy(tmp, tr)
		closeErr := tmp.Close()
		if copyErr != nil {
			os.Remove(tmpName)
			return copyErr
		}
		if closeErr != nil {
			os.Remove(tmpName)
			return closeErr
		}
		if err := os.Chmod(tmpName, 0o755); err != nil {
			os.Remove(tmpName)
			return err
		}
		if err := os.Rename(tmpName, exe); err != nil {
			os.Remove(tmpName)
			return err
		}
		return nil
	}
}

// checkWritable confirms that we can create and remove a file in dir.
// On Linux, replacing a running binary works because os.Rename only needs
// write+execute on the parent directory, not the file itself.
func checkWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".motd-write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}
