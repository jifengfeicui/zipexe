//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const swShownormal = 1

var (
	shell32       = syscall.NewLazyDLL("shell32.dll")
	shellExecuteW = shell32.NewProc("ShellExecuteW")
	isUserAnAdmin = shell32.NewProc("IsUserAnAdmin")
)

func ensureElevated() (bool, error) {
	ret, _, _ := isUserAnAdmin.Call()
	if ret != 0 {
		return false, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	verb, err := syscall.UTF16PtrFromString("runas")
	if err != nil {
		return false, err
	}
	file, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return false, err
	}
	params, err := syscall.UTF16PtrFromString(joinWindowsArgs(os.Args[1:]))
	if err != nil {
		return false, err
	}
	dir, err := syscall.UTF16PtrFromString(cwd)
	if err != nil {
		return false, err
	}

	code, _, callErr := shellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		uintptr(unsafe.Pointer(dir)),
		swShownormal,
	)
	if code <= 32 {
		return false, fmt.Errorf("request elevation failed: %v, code=%d", callErr, code)
	}
	return true, nil
}

func joinWindowsArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, quoteWindowsArg(arg))
	}
	return strings.Join(quoted, " ")
}

func quoteWindowsArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\n\v\"") {
		return arg
	}

	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for _, r := range arg {
		switch r {
		case '\\':
			backslashes++
		case '"':
			b.WriteString(strings.Repeat("\\", backslashes*2+1))
			b.WriteRune(r)
			backslashes = 0
		default:
			if backslashes > 0 {
				b.WriteString(strings.Repeat("\\", backslashes))
				backslashes = 0
			}
			b.WriteRune(r)
		}
	}
	if backslashes > 0 {
		b.WriteString(strings.Repeat("\\", backslashes*2))
	}
	b.WriteByte('"')
	return b.String()
}
