package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const (
	githubOwner = "stackfox-labs"
	githubRepo  = "luumen"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func newSelfUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "self-update",
		Short:   "Update luu to the latest version",
		Long:    "Self-update fetches the latest luu release from GitHub and replaces the current binary.",
		Args:    requireNoPositionalArgs(),
		GroupID: "other",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSelfUpdate(cmd)
		},
	}
}

func runSelfUpdate(cmd *cobra.Command) error {
	writer := cmd.OutOrStdout()

	statusf(cmd, "Checking for updates...")

	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := currentVersion()

	if current != "dev" && current == latest {
		fmt.Fprintf(writer, "%s luu is already up to date %s\n",
			successPrefix(writer), styleMuted(writer, "("+current+")"))
		return nil
	}

	assetName := selfUpdateAssetName()

	var downloadURL, checksumsURL string
	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			downloadURL = asset.BrowserDownloadURL
		case "checksums.txt":
			checksumsURL = asset.BrowserDownloadURL
		}
	}

	if downloadURL == "" {
		return fmt.Errorf(
			"no release asset found for %s/%s Next: check https://github.com/%s/%s/releases/%s for available assets",
			runtime.GOOS, runtime.GOARCH, githubOwner, githubRepo, release.TagName,
		)
	}

	statusf(cmd, "Downloading %s...", styleAccent(writer, release.TagName))

	data, err := fetchBytes(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if checksumsURL != "" {
		statusf(cmd, "Verifying checksum...")
		if err := verifyChecksum(data, assetName, checksumsURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}

	if err := extractAndReplace(data, execPath); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Fprintf(writer, "%s updated to %s\n", successPrefix(writer), styleAccent(writer, release.TagName))
	return nil
}

func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "luu/"+currentVersion())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("could not parse release response: %w", err)
	}
	return &release, nil
}

func fetchBytes(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download interrupted: %w", err)
	}
	return data, nil
}

// verifyChecksum parses checksums.txt (format: "sha256:<hex>  <filename>" per line)
// and confirms the SHA-256 of data matches the entry for assetName.
func verifyChecksum(data []byte, assetName, checksumsURL string) error {
	checksumData, err := fetchBytes(checksumsURL)
	if err != nil {
		return fmt.Errorf("could not fetch checksums.txt: %w", err)
	}

	expected, err := parseChecksums(string(checksumData), assetName)
	if err != nil {
		return err
	}

	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])

	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", expected, got)
	}
	return nil
}

// parseChecksums finds the SHA-256 hex for name in a checksums.txt file.
// Supported line formats:
//
//	sha256:<hex>  <name>
//	<hex>  <name>
func parseChecksums(content, name string) (string, error) {
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Strip optional "sha256:" prefix.
		line = strings.TrimPrefix(line, "sha256:")

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		if parts[1] == name {
			return strings.ToLower(parts[0]), nil
		}
	}
	return "", fmt.Errorf("no checksum entry found for %s", name)
}

func selfUpdateAssetName() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("luu-%s-%s.zip", runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("luu-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
}

func extractAndReplace(data []byte, dest string) error {
	var (
		binary []byte
		err    error
	)
	if runtime.GOOS == "windows" {
		binary, err = extractFromZip(data, "luu.exe")
	} else {
		binary, err = extractFromTarGz(data, "luu")
	}
	if err != nil {
		return err
	}
	return replaceBinary(binary, dest)
}

func extractFromZip(data []byte, target string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("could not open zip archive: %w", err)
	}

	for _, f := range r.File {
		if f.Name == target {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("could not open %s in archive: %w", target, err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("%s not found in archive", target)
}

func extractFromTarGz(data []byte, target string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("could not decompress archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not read archive: %w", err)
		}
		if hdr.Name == target {
			return io.ReadAll(tr)
		}
	}

	return nil, fmt.Errorf("%s not found in archive", target)
}

func replaceBinary(binary []byte, dest string) error {
	tmp, err := os.CreateTemp("", "luu-update-*")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(binary); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("could not write binary: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("could not mark binary as executable: %w", err)
	}

	// Windows locks running executables, so rename the old one out of the way first.
	if runtime.GOOS == "windows" {
		oldPath := dest + ".old"
		_ = os.Remove(oldPath) // clean up leftover from a previous update
		if err := os.Rename(dest, oldPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("could not rename current binary: %w", err)
		}
		if err := os.Rename(tmpPath, dest); err != nil {
			_ = os.Rename(oldPath, dest) // attempt restore
			os.Remove(tmpPath)
			return fmt.Errorf("could not move new binary into place: %w", err)
		}
		// The .old file stays locked by Windows until the process exits; cleaned up on next update.
		return nil
	}

	return os.Rename(tmpPath, dest)
}
