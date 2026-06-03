//go:build !windows

package main

func ensureElevated() (bool, error) {
	return false, nil
}
