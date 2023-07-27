package utils

import "runtime"

//goland:noinspection GoBoolExpressions
const (
	IsWindows = runtime.GOOS == "windows"
	IsLinux   = runtime.GOOS == "linux"
	IsDarwin  = runtime.GOOS == "darwin"
)
