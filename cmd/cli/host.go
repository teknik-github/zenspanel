package main

import "os"

// osHostname is split out so model.go doesn't pull os into its imports —
// keeps the model file focused on bubbletea wiring.
func osHostname() (string, error) {
	return os.Hostname()
}
