package osdetect

import (
	"bufio"
	"os"
	"os/exec"
	"strings"

	"ZHC-project/internal/guest/shared"
)

var TrackedTools = []string{
	"nmap", "nikto", "gobuster", "hydra", "sqlmap", "wpscan", "dirb", "msfconsole",
	"wfuzz", "burpsuite", "john", "hashcat", "aircrack-ng", "ettercap", "snort",
	"tcpdump", "tshark", "netcat", "gvm-start", "omp", "maltego", "recon-ng",
	"theharvester", "dnsenum", "dnsmap", "dnsrecon", "fierce", "sublist3r",
	"amass", "shodan", "censys", "zoomeye", "masscan", "unicornscan",
}

func DetectSystemInfo() (*shared.SystemInfo, error) {
	info := &shared.SystemInfo{
		OSName: "Unknown Linux",
		AvailableTools: make([]string, 0),
	}

	file, err := os.Open("/etc/os-release")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "NAME=") {
				info.OSName = strings.Trim(strings.TrimPrefix(line, "NAME="), `"`)
			} else if strings.HasPrefix(line, "VERSION_ID=") {
				info.OSVersion = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
			}
		}
	}

	osLower := strings.ToLower(info.OSName)
	if strings.Contains(osLower, "kali") || strings.Contains(osLower, "parrot") || strings.Contains(osLower, "blackarch") {
		info.IsPentestDistro = true
	}

	for _, tool := range TrackedTools {
		if _, err := exec.LookPath(tool); err == nil {
			info.AvailableTools = append(info.AvailableTools, tool)
		}
	}
	return info, nil
}