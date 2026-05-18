package main

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary = lipgloss.Color("#6366f1") // indigo-500
	colorSuccess = lipgloss.Color("#10b981") // emerald-500
	colorError   = lipgloss.Color("#ef4444") // red-500
	colorMuted   = lipgloss.Color("#9ca3af") // gray-400
	colorAccent  = lipgloss.Color("#f59e0b") // amber-500

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	subtleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	successStyle = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	accentStyle  = lipgloss.NewStyle().Foreground(colorAccent)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(1, 2)
)
