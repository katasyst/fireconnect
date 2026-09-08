package fileutil

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path atomically via a temp file in the same directory.
// When mode is 0, existing file permissions are preserved.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	targetMode := mode
	if targetMode == 0 {
		if info, err := os.Stat(path); err == nil {
			targetMode = info.Mode().Perm()
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	randBytes := make([]byte, 6)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Errorf("generate temp file name: %w", err)
	}
	tempPath := filepath.Join(dir, fmt.Sprintf(".%s.%s.tmp", filepath.Base(path), hex.EncodeToString(randBytes)))

	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if targetMode != 0 {
		f, err := os.OpenFile(tempPath, flags, targetMode)
		if err != nil {
			return err
		}
		if _, err := f.Write(data); err != nil {
			_ = f.Close()
			_ = os.Remove(tempPath)
			return err
		}
		if err := f.Close(); err != nil {
			_ = os.Remove(tempPath)
			return err
		}
	} else {
		if err := os.WriteFile(tempPath, data, 0o644); err != nil {
			_ = os.Remove(tempPath)
			return err
		}
	}

	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	if targetMode != 0 {
		if err := os.Chmod(path, targetMode); err != nil {
			return err
		}
	}
	return nil
}
