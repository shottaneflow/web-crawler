package utilsCraw

import (
	"os/exec"
	"runtime"
)

func FilterLinks(links []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, link := range links {
		if link != "" && !seen[link] {
			seen[link] = true
			result = append(result, link)
		}
	}
	return result
}

func ForceKillChrome() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("taskkill", "/IM", "chrome.exe", "/F")
	} else {
		cmd = exec.Command("pkill", "-f", "chrome|chromium")
	}
	_ = cmd.Run()
}
