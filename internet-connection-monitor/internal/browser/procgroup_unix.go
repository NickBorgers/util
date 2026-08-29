//go:build unix

package browser

import (
	"errors"
	"os/exec"
	"syscall"
)

// setProcessGroupAndGroupKill configures the Chrome exec.Cmd so that:
//  1. Chrome (and everything it forks - zygote, GPU process, one renderer
//     per navigation) runs in its own process group.
//  2. Cancelling the command's context kills that whole process group
//     instead of chromedp's default of SIGKILLing only the Chrome PID.
//
// Passing this via chromedp.ModifyCmdFunc replaces chromedp's own
// allocateCmdOptions (which just sets Pdeathsig on Linux - see
// allocate_linux.go in the chromedp module), so we reimplement that half
// too, Linux-only, to match.
//
// Without this, chromedp's default cancellation kills only the top-level
// Chrome process; its zygote/GPU/renderer children are left running,
// get re-parented onto this program once Chrome is gone, and are only
// cleaned up once internal/reaper's background wait4 loop catches up with
// them. Killing the whole group here makes cleanup immediate and
// deterministic instead of relying solely on the reaper.
func setProcessGroupAndGroupKill(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true

	setPdeathsig(cmd.SysProcAttr)

	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// Negative pid targets the whole process group (valid because
		// Setpgid above made Chrome its own group leader, pgid == pid).
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return err
		}
		return nil
	}
}
