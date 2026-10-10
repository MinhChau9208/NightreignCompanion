//go:build !windows

package helper

import (
	"context"
	"errors"
)

func Run(ctx context.Context, cfg Config) error {
	return errors.New("the measuring helper is only supported on Windows")
}
