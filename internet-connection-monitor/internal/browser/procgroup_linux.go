//go:build linux

package browser

import "syscall"

// setPdeathsig arranges for the Chrome process to be killed if this
// program dies before it can clean up. Mirrors chromedp's own default
// (see allocate_linux.go in the chromedp module), which we must
// reimplement because chromedp.ModifyCmdFunc replaces that default
// entirely rather than layering on top of it. Pdeathsig is Linux-specific.
func setPdeathsig(attr *syscall.SysProcAttr) {
	attr.Pdeathsig = syscall.SIGKILL
}
