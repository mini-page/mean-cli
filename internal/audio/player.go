package audio

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// PlayFromURL downloads the audio file from URL (mp3/wav) and plays it asynchronously.
func PlayFromURL(audioURL string) {
	if audioURL == "" {
		return
	}
	go func() {
		// Download file to temp dir
		tmpDir := os.TempDir()
		ext := ".mp3"
		if stringsHasSuffix(audioURL, ".wav") {
			ext = ".wav"
		}
		tempFile := filepath.Join(tmpDir, fmt.Sprintf("mean_pronounce_%d%s", time.Now().UnixNano(), ext))

		if err := downloadFile(audioURL, tempFile); err != nil {
			return
		}
		defer os.Remove(tempFile)

		// Play based on platform
		_ = playLocalFile(tempFile)
	}()
}

func downloadFile(url, filepath string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func playLocalFile(file string) error {
	switch runtime.GOOS {
	case "windows":
		// Play MP3 using Windows Media Player through PowerShell
		psCommand := fmt.Sprintf(
			`Add-Type -AssemblyName PresentationCore; $m = New-Object System.Windows.Media.MediaPlayer; $m.Open('%s'); $m.Play(); Start-Sleep -Seconds 4`,
			file,
		)
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCommand)
		return cmd.Run()

	case "darwin":
		// MacOS native CLI audio player
		cmd := exec.Command("afplay", file)
		return cmd.Run()

	default:
		// Linux players: try paplay, aplay, mpg123, or play in order
		players := []string{"paplay", "aplay", "mpg123", "play", "ffplay"}
		for _, p := range players {
			if path, err := exec.LookPath(p); err == nil {
				var cmd *exec.Cmd
				if p == "ffplay" {
					cmd = exec.Command(path, "-nodisp", "-autoexit", file)
				} else {
					cmd = exec.Command(path, file)
				}
				if err := cmd.Run(); err == nil {
					return nil
				}
			}
		}
		return fmt.Errorf("no audio player found on Linux")
	}
}

// Simple string suffix helper to avoid importing "strings" in basic signature
func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
