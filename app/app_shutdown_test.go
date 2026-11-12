package app_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/go-faster/sdk/app"
)

// The shutdown path ends in os.Exit, so it cannot be exercised in-process: the test binary re-execs
// itself as the application, signals it for real, and reads what it printed. These markers are the
// protocol between the two.
const (
	helperEnv     = "GO_FASTER_SDK_APP_HELPER"
	markerStarted = "app-helper-started"
	markerStopped = "app-helper-observed-shutdown"
)

// TestAppHelper is the application half of TestRunShutsDownOnSignal. It is a no-op unless the
// parent asks for it through the environment.
func TestAppHelper(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		t.Skip("helper process, driven by TestRunShutsDownOnSignal")
	}

	app.Run(func(ctx context.Context, _ *zap.Logger, _ *app.Telemetry) error {
		fmt.Println(markerStarted)
		<-ctx.Done()
		fmt.Println(markerStopped)

		return nil
	})
}

// TestRunShutsDownOnSignal asserts Run cancels its context on both signals a process is stopped
// with. SIGTERM is the one a container runtime sends, and the Go runtime kills the process outright
// unless it is watched — so without it an application never reaches its shutdown path.
func TestRunShutsDownOnSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX signals are not deliverable on Windows")
	}

	for _, tt := range []struct {
		name   string
		signal syscall.Signal
	}{
		{"SIGINT", syscall.SIGINT},
		{"SIGTERM", syscall.SIGTERM},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()

			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestAppHelper", "-test.v")
			cmd.Env = append(os.Environ(),
				helperEnv+"=1",
				// Keep the helper off the network: the telemetry exporters would otherwise dial
				// their default endpoints and slow the test down.
				"OTEL_TRACES_EXPORTER=none",
				"OTEL_METRICS_EXPORTER=none",
				"OTEL_LOGS_EXPORTER=none",
			)

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatalf("stdout pipe: %v", err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatalf("start helper: %v", err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill() })

			lines := make(chan string, 32)
			go func() {
				defer close(lines)

				sc := bufio.NewScanner(stdout)
				for sc.Scan() {
					lines <- sc.Text()
				}
			}()

			// Signal only once the application is running: a signal delivered before Run installs
			// its handler would kill the helper and pass the test for the wrong reason.
			waitFor(t, lines, markerStarted)

			if err := cmd.Process.Signal(tt.signal); err != nil {
				t.Fatalf("signal helper: %v", err)
			}

			waitFor(t, lines, markerStopped)

			if err := cmd.Wait(); err != nil {
				t.Fatalf("helper exited with error: %v", err)
			}
		})
	}
}

// waitFor consumes lines until want appears, failing if the stream ends first.
func waitFor(t *testing.T, lines <-chan string, want string) {
	t.Helper()

	var seen []string
	for line := range lines {
		seen = append(seen, line)
		if line == want {
			return
		}
	}

	t.Fatalf("helper output ended before %q:\n%s", want, seen)
}
