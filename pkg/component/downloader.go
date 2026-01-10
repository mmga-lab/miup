package component

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a GitHub release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Downloader handles downloading components from GitHub
type Downloader struct {
	client    *http.Client
	userAgent string
}

// NewDownloader creates a new downloader
func NewDownloader() *Downloader {
	return &Downloader{
		client:    &http.Client{},
		userAgent: "miup/1.0",
	}
}

// GetLatestRelease fetches the latest release info from GitHub
func (d *Downloader) GetLatestRelease(ctx context.Context, repo string) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	return d.getRelease(ctx, url)
}

// GetRelease fetches a specific release by tag
func (d *Downloader) GetRelease(ctx context.Context, repo, tag string) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, tag)
	return d.getRelease(ctx, url)
}

func (d *Downloader) getRelease(ctx context.Context, url string) (*GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", d.userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("release not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}
	return &release, nil
}

// DownloadAsset downloads and extracts a release asset
func (d *Downloader) DownloadAsset(ctx context.Context, asset *Asset, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", d.userAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Create progress bar only if stderr is a terminal (TTY)
	// In non-TTY environments (e.g., CI, piped output), progressbar produces
	// excessive output that can cause issues
	var reader io.Reader
	if term.IsTerminal(int(os.Stderr.Fd())) {
		bar := progressbar.NewOptions64(
			asset.Size,
			progressbar.OptionSetDescription(fmt.Sprintf("Downloading %s", asset.Name)),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(40),
			progressbar.OptionShowCount(),
			progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "=",
				SaucerHead:    ">",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
		reader = io.TeeReader(resp.Body, bar)
	} else {
		// Non-TTY: just print a simple message
		fmt.Fprintf(os.Stderr, "Downloading %s (%d MB)...\n", asset.Name, asset.Size/1024/1024)
		reader = resp.Body
	}

	// Handle different archive types
	if strings.HasSuffix(asset.Name, ".tar.gz") || strings.HasSuffix(asset.Name, ".tgz") {
		return extractTarGz(reader, destDir)
	}

	// Direct binary download
	destPath := filepath.Join(destDir, asset.Name)
	return downloadToFile(reader, destPath)
}

// extractTarGz extracts a tar.gz archive to the destination directory
func extractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		target := filepath.Join(destDir, header.Name)

		// Security: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to extract file: %w", err)
			}
			f.Close()
		}
	}
	return nil
}

// downloadToFile downloads content directly to a file
func downloadToFile(r io.Reader, destPath string) error {
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// FindAsset finds the appropriate asset for the current platform
func FindAsset(release *GitHubRelease, assetNameFunc func(version, os, arch string) string) (*Asset, error) {
	expectedName := assetNameFunc(release.TagName, runtime.GOOS, runtime.GOARCH)

	for i := range release.Assets {
		if release.Assets[i].Name == expectedName {
			return &release.Assets[i], nil
		}
	}

	// List available assets for debugging
	var available []string
	for _, a := range release.Assets {
		available = append(available, a.Name)
	}

	return nil, fmt.Errorf("no asset found for %s/%s, expected: %s\navailable assets: %v",
		runtime.GOOS, runtime.GOARCH, expectedName, available)
}
