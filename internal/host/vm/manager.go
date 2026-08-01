package vm

import (
	"fmt"
	"os"
	"os/exec"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"
)

type VMManager struct {
	conn *libvirt.Connect
	dom *libvirt.Domain
	overlayPath string
	viewer *SPICEViewer
}

func NewVMManager() (*VMManager, error) {
	conn,  err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to qemu:///system: %w", err)
	}
	return &VMManager{conn: conn}, nil
}

func (m *VMManager) CreateOverlay(baseImage, overlayPath string) error {
	m.overlayPath = overlayPath

	_ = os.Remove(overlayPath)

	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-b", baseImage, "-F", "qcow2", overlayPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create overlay image: %w", err)
	}
	return nil
}

func (m *VMManager) StartVM(xmlPath string) error {
	xmlBytes, err := os.ReadFile(xmlPath)
	if err != nil {
		return fmt.Errorf("failed to read VM XML config file: %w", err)
	}

	dom, err := m.conn.DomainCreateXML(string(xmlBytes), 0)
	if err != nil {
		return fmt.Errorf("failed to create domain from xml: %w", err)
	}
	m.dom = dom

	spicePort, err := m.getSPICEPort()
	if err != nil {
		return fmt.Errorf("failed to extract spice port: %w", err)
	}

	if spicePort == 0 {
		spicePort = 5900
		fmt.Printf("SPICE port not found in xml, falling back to the default %d\n", spicePort)
	}

	viewer, err := LaunchSPICEViewer(spicePort)
	if err != nil {
		fmt.Printf("Warning: failed to launch SPICE viewer: %v\n", err)
	} else {
		m.viewer = viewer
		fmt.Printf("SPICE viewer launched on port %d\n", spicePort)
	}

	return nil
}

func (m *VMManager) getSPICEPort() (int, error) {
	xmlDesc, err := m.dom.GetXMLDesc(libvirt.DomainXMLFlags(libvirt.DOMAIN_XML_SECURE))
	if err != nil {
		return 0, err
	}

	domCfg := &libvirtxml.Domain{}
	if err := domCfg.Unmarshal(xmlDesc); err != nil {
		return 0, err
	}

	for _, dev := range domCfg.Devices.Graphics {
		if dev.Spice != nil && dev.Spice.Port != 0 {
			return dev.Spice.Port, nil
		}
	}
	return 0, fmt.Errorf("SPICE port not found in domain XML")
}

func (m *VMManager) StopVM() error {
	if m.viewer != nil {
		_ = m.viewer.Stop()
	}

	if m.dom != nil {
		_ = m.dom.Destroy()
		_ = m.dom.Free()
		m.dom = nil
	}

	if m.conn != nil {
		_, _ = m.conn.Close()
	}

	if m.overlayPath != "" {
		_ = os.Remove(m.overlayPath)
	}

	return nil
}