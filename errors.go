package configresolver

import "errors"

var (
	ErrCircularReference = errors.New("circular reference detected")
	ErrUnknownReference  = errors.New("unknown gitlab reference")
)
