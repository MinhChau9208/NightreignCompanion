//go:build !windows

package ocr

import "image"

type Engine struct{}

func New(lang string) (*Engine, error) { return nil, errUnsupported }

func (e *Engine) Recognize(img *image.Gray) ([]string, error) { return nil, errUnsupported }

func (e *Engine) Close() {}
