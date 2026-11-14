//go:build windows

package agent

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Embed the smartmontools directory so the agent builds without the binaries staged.
// When go generate downloads smartctl.exe it is embedded too; SMART stays optional.
//go:embed all:smartmontools
var smartctlFs embed.FS

var (
	smartctlOnce sync.Once
	smartctlPath string
	smartctlErr  error
)

func ensureEmbeddedSmartctl() (string, error) {
	smartctlOnce.Do(func() {
		destDir := filepath.Join(os.TempDir(), "beszel", "smartmontools")
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			smartctlErr = fmt.Errorf("failed to create smartctl directory: %w", err)
			return
		}

		data, err := smartctlFs.ReadFile("smartmontools/smartctl.exe")
		if err != nil {
			smartctlErr = fmt.Errorf("embedded smartctl not available (run go generate ./agent): %w", err)
			return
		}

		destPath := filepath.Join(destDir, "smartctl.exe")
		if err := os.WriteFile(destPath, data, 0o755); err != nil {
			smartctlErr = fmt.Errorf("failed to write embedded smartctl: %w", err)
			return
		}

		smartctlPath = destPath
	})

	return smartctlPath, smartctlErr
}
