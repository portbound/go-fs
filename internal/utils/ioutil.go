// Package utils
package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type chanl struct {
	bytesWritten int64
	err          error
}

func StageFileToDisk(ctx context.Context, path string, fileName string, reader io.Reader) (string, int64, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", 0, fmt.Errorf("util.StageFileToDisk: failed to create storage dir at '%s': %w", path, err)
	}

	file, err := os.Create(filepath.Join(path, fileName))
	if err != nil {
		return "", 0, fmt.Errorf("util.StageFileToDisk: failed to create temp file: %w", err)
	}
	defer file.Close()

	ch := make(chan *chanl, 1)
	go func() {
		bytesWritten, copyErr := io.Copy(file, reader)
		ch <- &chanl{bytesWritten: bytesWritten, err: copyErr}
	}()

	select {
	case <-ctx.Done():
		os.Remove(file.Name())
		return "", 0, ctx.Err()
	case result := <-ch:
		if result.err != nil {
			os.Remove(file.Name())
			return "", 0, fmt.Errorf("util.StageFileToDisk: failed to write to tmp file: %w", result.err)
		}
		return file.Name(), result.bytesWritten, nil
	}
}
