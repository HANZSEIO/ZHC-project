package vm

import (
	"fmt"
	"os/exec"
)

type SPICEViewer struct {
	cmd *exec.Cmd
}


func LaunchSPICEViewer(port int) (*SPICEViewer, error) {
	uri := fmt.Sprintf("spice://127.0.0.1:%d", port)

	if _, err := exec.LookPath("remote-viewer"); err != nil {
		return nil, fmt.Errorf("remote-viewer not found in PATH: %w", err)
	}

	cmd := exec.Command("remote-viewer", uri)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to launch remote-viewer: %w", err)
	}

	return &SPICEViewer{cmd: cmd}, nil
}

func (v *SPICEViewer) Stop() error {
	if v.cmd != nil && v.cmd.Process != nil {
		return v.cmd.Process.Kill()
	}
	return nil
}