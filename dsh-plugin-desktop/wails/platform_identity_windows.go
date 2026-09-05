//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func setWindowsAppUserModelID(appID string) error {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
	ptr, err := syscall.UTF16PtrFromString(appID)
	if err != nil {
		return err
	}
	r0, _, e1 := proc.Call(uintptr(unsafe.Pointer(ptr)))
	if r0 != 0 {
		if e1 != nil {
			return fmt.Errorf("SetCurrentProcessExplicitAppUserModelID: %w", e1)
		}
		return fmt.Errorf("SetCurrentProcessExplicitAppUserModelID HRESULT=0x%x", r0)
	}
	return nil
}
