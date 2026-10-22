//go:build windows

package command

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

const taskkillTimeout = 10 * time.Second

func killTreeOnCancel(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), taskkillTimeout)
		defer cancel()

		kill := exec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		if err := kill.Run(); err != nil {
			return errors.Join(fmt.Errorf("taskkill could not stop the process tree: %w", err), cmd.Process.Kill())
		}
		return nil
	}
}
