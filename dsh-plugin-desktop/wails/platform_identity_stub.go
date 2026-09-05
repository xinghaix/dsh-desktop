//go:build !windows

package main

func setWindowsAppUserModelID(appID string) error {
	_ = appID
	return nil
}
