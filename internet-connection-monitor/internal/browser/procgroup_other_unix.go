//go:build unix && !linux

package browser

import "syscall"

// setPdeathsig is a no-op outside Linux: Pdeathsig is a Linux-specific
// SysProcAttr field, and chromedp's own default only sets it on Linux
// too (see allocate_other.go in the chromedp module). Process-group
// cleanup via cmd.Cancel still applies on every unix platform.
func setPdeathsig(attr *syscall.SysProcAttr) {}
