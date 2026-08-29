//go:build unix

// Package reaper reaps orphaned child processes so they don't accumulate
// as zombies.
//
// This binary is the container's PID 1 (see the project Dockerfile:
// `ENTRYPOINT ["/app/internet-monitor"]`, no tini and no `--init`). Every
// probe launches headless Chrome via chromedp
// (internal/browser/controller_impl.go), and Chrome itself forks a zygote
// plus per-navigation GPU and renderer processes. chromedp's ExecAllocator
// waits on and reaps only the single Chrome process it directly started —
// Chrome's own children are never something it knows about, let alone
// waits for.
//
// When the allocator context is cancelled (on success, on error, or on
// timeout), only the top-level Chrome process is killed. Its children lose
// their parent, and because there is no init process in the container they
// are re-parented directly onto this process. Nothing was ever waiting on
// them, so once each one exits it becomes a permanent zombie: one leaked
// "<defunct>" entry per Chrome launch, accumulating without bound until the
// container's cgroup PID limit is hit and every subsequent fork() in the
// container starts failing.
//
// Start installs a background goroutine that reaps *any* exited child of
// this process — known or not — for the lifetime of the program, exactly
// as an init process would.
package reaper

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// Start installs a SIGCHLD-driven reaper. It runs until the process exits;
// there is no corresponding Stop, since a PID 1 process must always be
// ready to reap.
func Start(logger *slog.Logger) {
	sigChan := make(chan os.Signal, 8)
	signal.Notify(sigChan, syscall.SIGCHLD)

	go func() {
		// Drain anything that exited before the handler was installed
		// (e.g. between process start and this goroutine scheduling),
		// then reap on every subsequent SIGCHLD.
		reapAll(logger)
		for range sigChan {
			reapAll(logger)
		}
	}()
}

// reapAll drains every currently-exited child via non-blocking wait4,
// looping until there is nothing left. A single SIGCHLD delivery can
// coalesce multiple child exits (Unix signals are not queued), so one
// wait4 call per signal is not sufficient — this must loop until wait4
// reports no more exited children.
func reapAll(logger *slog.Logger) {
	for {
		var status syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		if err != nil {
			// ECHILD: no children (reaped or otherwise) exist at all.
			// Any other error: nothing more we can do this pass.
			return
		}
		if pid <= 0 {
			// pid == 0: WNOHANG and no child has exited right now.
			return
		}
		if logger != nil {
			logger.Debug("reaper: reaped child process",
				"pid", pid,
				"exit_status", status.ExitStatus(),
				"signaled", status.Signaled(),
			)
		}
	}
}
