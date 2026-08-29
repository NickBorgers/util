//go:build windows

package browser

import "os/exec"

// setProcessGroupAndGroupKill is a no-op on Windows, which has no POSIX
// process groups or SIGKILL. Falls back to chromedp's own default
// cancellation behavior (kill the Chrome process only). This project only
// ships as a Linux container (see the Dockerfile,
// chromedp/headless-shell), so this exists purely so the package still
// builds for local tooling on Windows.
func setProcessGroupAndGroupKill(cmd *exec.Cmd) {}
