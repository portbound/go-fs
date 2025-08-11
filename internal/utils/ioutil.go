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
	if err != nil {
		return "", fmt.Errorf("util.StageFileToDisk: failed to create temp file: %w", err)
	}
	defer file.Close()

	ch := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(file, reader)
		ch <- copyErr
	}()

	select {
	case <-ctx.Done():
		os.Remove(file.Name())
		return "", ctx.Err()
	case chanErr := <-ch:
		if chanErr != nil {
			os.Remove(file.Name())
			return "", fmt.Errorf("util.StageFileToDisk: failed to write to tmp file: %w", chanErr)
		}
		return file.Name(), nil
	}
}
