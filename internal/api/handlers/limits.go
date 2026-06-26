package handlers

import "fmt"

func enforceLimit(current, max int, name string) error {
	if max > 0 && current >= max {
		return fmt.Errorf("%s quota reached", name)
	}
	return nil
}
