package util

import (
	"context"
	"fmt"
	"io"
	"os"
)

func StageFileToDisk(ctx context.Context, path string, pattern string, reader io.Reader) error {
	type result struct {
		bytes int64
		err   error
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("util.StageFileToDisk: failed to create storage dir at '%s': %w", path, err)
	}

	file, err := os.CreateTemp(path, pattern)
	if err != nil {
		return fmt.Errorf("util.StageFileToDisk: failed to create temp file: %w", err)
	}
	defer file.Close()

	ch := make(chan result, 1)
	go func() {
		bytes, err := io.Copy(file, reader)
		ch <- result{bytes: bytes, err: err}
	}()

	select {
	case <-ctx.Done():
		os.Remove(path)
		return ctx.Err()
	case result := <-ch:
		if result.err != nil {
			os.Remove(path)
			return fmt.Errorf("util.StageFileToDisk: failed to write to tmp file: %w", result.err)
		}
	}

	return nil
}
