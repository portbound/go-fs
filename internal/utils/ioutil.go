// Package utils
package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func StageFileToDisk(ctx context.Context, path string, fileName string, reader io.Reader) (string, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("util.StageFileToDisk: failed to create storage dir at '%s': %w", path, err)
	}

	file, err := os.Create(filepath.Join(path, fileName))
	// file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("util.StageFileToDisk: failed to create temp file: %w", err)
	}
	defer file.Close()

	ch := make(chan error, 1)
	go func() {
		_, chErr := io.Copy(file, reader)
		ch <- chErr
	}()

	select {
	case <-ctx.Done():
		os.Remove(path)
		return "", ctx.Err()
	case result := <-ch:
		if result != nil {
			os.Remove(path)
			return "", fmt.Errorf("util.StageFileToDisk: failed to write to tmp file: %w", err)
		}
		return file.Name(), nil
	}
}
