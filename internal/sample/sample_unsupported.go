//go:build !darwin && !linux

package sample

import (
	"context"
	"fmt"
	"runtime"
)

func collect(ctx context.Context, live bool) (Snapshot, error) {
	_ = ctx
	_ = live
	return Snapshot{}, fmt.Errorf("depths: unsupported platform %s (supported: darwin, linux)", runtime.GOOS)
}
