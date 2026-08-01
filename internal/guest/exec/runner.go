package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"ZHC-project/internal/guest/shared"
)

const DefaultTimeoutSec = 300

func ExecuteTool(req shared.ExecRequest) shared.ExecResult {
	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = DefaultTimeoutSec
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	binPath, err := exec.LookPath(req.ToolName)
	if err != nil {
		return shared.ExecResult{
			Success:  false,
			ExitCode: -1,
			ErrorMsg: fmt.Sprintf("Tool '%s' not found in PATH: %v", req.ToolName, err),
		}
	}
	cmd := exec.CommandContext(ctx, binPath, req.Args...)

	var stdoutBuf, sderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &sderrBuf

	err = cmd.Run()

	result := shared.ExecResult{
		Stdout: stdoutBuf.String(),
		Stderr: sderrBuf.String(),
	}

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Success = false
			result.ExitCode = -1
			result.ErrorMsg = fmt.Sprintf("Execution timed out after %d seconds", timeout)
			return result
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			result.ErrorMsg = fmt.Sprintf("Command exited with code %d", result.ExitCode)
		} else {
			result.ExitCode = -1
			result.ErrorMsg = err.Error()
		}

		result.Success = false
		return result
	}

	result.Success = true
	result.ExitCode = 0
	return result
}
