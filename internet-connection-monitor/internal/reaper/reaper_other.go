//go:build !unix

package reaper

import "log/slog"

// Start is a no-op on platforms without POSIX SIGCHLD/wait4 semantics
// (Windows). The zombie-process leak this package fixes is specific to
// running as PID 1 in a Linux container.
func Start(logger *slog.Logger) {}
