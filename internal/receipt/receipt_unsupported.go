//go:build !darwin && !linux

package receipt

import (
	"fmt"
	"runtime"
)

func Dir() (string, error) {
	return "", fmt.Errorf("depths: receipts unsupported on %s", runtime.GOOS)
}
