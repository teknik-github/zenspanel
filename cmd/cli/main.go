package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const cliVersion = "1.0.0"

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "zenspanel-cli must be run as root (config and socket require it)")
		os.Exit(1)
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(newRootModel(cfg), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
