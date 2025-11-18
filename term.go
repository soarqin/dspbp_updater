package main

import (
	"os"
	"syscall"

	"github.com/pterm/pterm"
)

const (
	// Windows console mode flags
	ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	STD_OUTPUT_HANDLE                  = ^uintptr(0) - 11 + 1
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

func init() {
	if supportsColor() {
		pterm.EnableColor()
	} else {
		pterm.DisableColor()
	}
}

// EnableWindowsANSI enables ANSI escape sequence processing on Windows
func enableWindowsANSI() bool {
	handle, err := syscall.Open("CONOUT$", syscall.O_RDWR, 0)
	if err != nil || handle == syscall.InvalidHandle {
		handle = syscall.Stdout
	} else {
		defer syscall.Close(syscall.Handle(handle))
	}

	var mode uint32
	if err := syscall.GetConsoleMode(handle, &mode); err != nil {
		return false
	}

	// Enable virtual terminal processing
	mode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if ret, _, _ := procSetConsoleMode.Call(uintptr(handle), uintptr(mode)); ret == 0 {
		return false
	}

	return true
}

func supportsColor() bool {
	// Check NO_COLOR first
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check FORCE_COLOR
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// On Windows, try to enable ANSI support
	if enableWindowsANSI() {
		return true
	}

	// Fallback checks
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}

	// Check for known color-supporting environments
	if os.Getenv("WT_SESSION") != "" || // Windows Terminal
		os.Getenv("ConEmuANSI") == "ON" || // ConEmu
		os.Getenv("COLORTERM") != "" {
		return true
	}

	return false
}
