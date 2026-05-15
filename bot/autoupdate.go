package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	RepoOwner   = "akikohatsune"
	RepoName    = "teto"
	GitHubAPI   = "https://api.github.com/repos/" + RepoOwner + "/" + RepoName + "/releases/latest"
	CurrentVersion = "1.0.0" // Update this with each release
)

type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		ContentType        string `json:"content_type"`
	} `json:"assets"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
}

// CheckForUpdates checks GitHub for new releases
func (b *TetoBot) CheckForUpdates() (*GitHubRelease, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(GitHubAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %v", err)
	}

	return &release, nil
}

// CompareVersions returns true if newVersion > currentVersion
func CompareVersions(current, latest string) bool {
	// Simple version comparison: v1.2.3 -> 1.2.3
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	currentParts := strings.Split(current, ".")
	latestParts := strings.Split(latest, ".")

	for i := 0; i < len(currentParts) && i < len(latestParts); i++ {
		var curNum, latestNum int
		fmt.Sscanf(currentParts[i], "%d", &curNum)
		fmt.Sscanf(latestParts[i], "%d", &latestNum)

		if latestNum > curNum {
			return true
		}
		if curNum > latestNum {
			return false
		}
	}

	return len(latestParts) > len(currentParts)
}

// DownloadRelease downloads the appropriate binary for the current OS
func DownloadRelease(release *GitHubRelease, downloadDir string) (string, error) {
	if len(release.Assets) == 0 {
		return "", fmt.Errorf("no assets found in release")
	}

	os.MkdirAll(downloadDir, 0755)

	// Determine the appropriate binary to download
	var targetAsset string
	osName := runtime.GOOS   // linux, darwin, windows
	arch := runtime.GOARCH   // amd64, arm64, etc.
	execExt := ""
	if osName == "windows" {
		execExt = ".exe"
	}

	searchPattern := fmt.Sprintf("%s-%s%s", osName, arch, execExt)

	for _, asset := range release.Assets {
		if strings.Contains(strings.ToLower(asset.Name), strings.ToLower(searchPattern)) {
			targetAsset = asset.BrowserDownloadURL
			break
		}
	}

	if targetAsset == "" {
		// Fallback: try to find any binary for the OS
		for _, asset := range release.Assets {
			if strings.Contains(strings.ToLower(asset.Name), osName) && !strings.Contains(asset.Name, ".tar") && !strings.Contains(asset.Name, ".zip") {
				targetAsset = asset.BrowserDownloadURL
				break
			}
		}
	}

	if targetAsset == "" {
		return "", fmt.Errorf("no compatible binary found for %s/%s", osName, arch)
	}

	// Download the binary
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(targetAsset)
	if err != nil {
		return "", fmt.Errorf("failed to download release: %v", err)
	}
	defer resp.Body.Close()

	fileName := filepath.Base(targetAsset)
	filePath := filepath.Join(downloadDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	// Make executable on Unix-like systems
	if osName != "windows" {
		os.Chmod(filePath, 0755)
	}

	return filePath, nil
}

// ApplyUpdate replaces the current binary with the new one
func ApplyUpdate(newBinaryPath string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %v", err)
	}

	// Create backup
	backupPath := executable + ".bak"
	if err := os.Rename(executable, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// Move new binary to current location
	if err := os.Rename(newBinaryPath, executable); err != nil {
		// Restore backup if move fails
		os.Rename(backupPath, executable)
		return fmt.Errorf("failed to apply update: %v", err)
	}

	// Make executable if on Unix
	if runtime.GOOS != "windows" {
		os.Chmod(executable, 0755)
	}

	return nil
}

// AutoUpdateLoop checks for updates periodically
func (b *TetoBot) AutoUpdateLoop(checkIntervalHours int) {
	if checkIntervalHours <= 0 {
		log.Println("[AutoUpdate] Disabled (CHECK_UPDATE_INTERVAL_HOURS <= 0)")
		return
	}

	ticker := time.NewTicker(time.Duration(checkIntervalHours) * time.Hour)
	defer ticker.Stop()

	log.Printf("[AutoUpdate] Checking for updates every %d hours", checkIntervalHours)

	for range ticker.C {
		b.PerformUpdateCheck()
	}
}

// PerformUpdateCheck performs a single update check
func (b *TetoBot) PerformUpdateCheck() {
	release, err := b.CheckForUpdates()
	if err != nil {
		log.Printf("[AutoUpdate] Failed to check for updates: %v", err)
		return
	}

	if CompareVersions(CurrentVersion, release.TagName) {
		log.Printf("[AutoUpdate] New version available: %s -> %s", CurrentVersion, release.TagName)
		
		// Notify owner
		if b.Settings.OwnerID != "" {
			msg := fmt.Sprintf("🚀 **Update Available!**\n\nNew version: `%s`\nChangelog:\n%s\n\nUpdate will be applied at next restart.", release.TagName, release.Body)
			b.Session.ChannelMessageSend(b.Settings.OwnerID, msg)
		}

		// Auto-download and apply
		downloadDir := filepath.Join(".", "updates")
		binaryPath, err := DownloadRelease(release, downloadDir)
		if err != nil {
			log.Printf("[AutoUpdate] Failed to download update: %v", err)
			return
		}

		log.Printf("[AutoUpdate] Update downloaded to: %s", binaryPath)
		
		// Optionally apply immediately or wait for restart
		// For safety, we just download and wait for next restart
		log.Printf("[AutoUpdate] Update ready. Will be applied on next bot restart.")

		if b.Settings.OwnerID != "" {
			b.Session.ChannelMessageSend(b.Settings.OwnerID, "✅ Update downloaded successfully! Restart the bot to apply.")
		}
	} else {
		log.Printf("[AutoUpdate] Already on latest version: %s", CurrentVersion)
	}
}

// ManualUpdate command - allows manual update trigger
func (b *TetoBot) TriggerManualUpdate() error {
	log.Println("[AutoUpdate] Starting manual update check...")
	
	release, err := b.CheckForUpdates()
	if err != nil {
		return err
	}

	if !CompareVersions(CurrentVersion, release.TagName) {
		return fmt.Errorf("already on latest version: %s", CurrentVersion)
	}

	downloadDir := filepath.Join(".", "updates")
	binaryPath, err := DownloadRelease(release, downloadDir)
	if err != nil {
		return err
	}

	log.Printf("[AutoUpdate] Downloaded: %s", binaryPath)
	
	// Apply update
	if err := ApplyUpdate(binaryPath); err != nil {
		return err
	}

	log.Printf("[AutoUpdate] Update applied! Please restart the bot.")
	
	// Restart bot
	executable, _ := os.Executable()
	cmd := exec.Command(executable)
	cmd.Start()
	
	// Give it time to start before closing
	time.Sleep(2 * time.Second)
	b.Close()
	os.Exit(0)

	return nil
}
