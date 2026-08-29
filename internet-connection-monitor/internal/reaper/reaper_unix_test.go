//go:build unix

package reaper

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// TestStart_ReapsUnwaitedChildren is a regression test for the zombie leak:
// a process that is exec'd via cmd.Start() but never explicitly cmd.Wait()'d
// (exactly what happens to Chrome's grandchildren once Chrome itself is
// killed and they're re-parented onto us) must not linger as a zombie once
// the reaper is running.
func TestStart_ReapsUnwaitedChildren(t *testing.T) {
	Start(nil)

	const n = 10
	pids := make([]int, 0, n)
	for i := 0; i < n; i++ {
		cmd := exec.Command("/bin/sh", "-c", "exit 0")
		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start child process: %v", err)
		}
		pids = append(pids, cmd.Process.Pid)
		// Deliberately do NOT call cmd.Wait() - simulates a process
		// whose exit this program never explicitly waits for.
	}

	// Give the reaper's goroutine a chance to receive SIGCHLD and drain
	// the exited children.
	deadline := time.Now().Add(5 * time.Second)
	for _, pid := range pids {
		for {
			// Wait4 on a specific PID with WNOHANG: if the process is
			// still an unreaped zombie, this returns immediately with
			// that pid and consumes it out from under the reaper (bad
			// for the test, but still proves it was reapable). If the
			// reaper already reaped it, the kernel no longer considers
			// it our child and this returns ECHILD - which is exactly
			// what we want to observe.
			var status syscall.WaitStatus
			wpid, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
			if err == syscall.ECHILD {
				break // reaped by the background reaper - success
			}
			if wpid == pid {
				t.Fatalf("pid %d was still an unreaped zombie %v after reaper ran (reaper did not reap it in time)", pid, time.Since(deadline.Add(-5*time.Second)))
			}
			if time.Now().After(deadline) {
				t.Fatalf("pid %d was not reaped within the deadline", pid)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}
