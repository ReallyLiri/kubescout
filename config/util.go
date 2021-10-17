package config

import (
	"fmt"
	"os"
	"strings"
)

func validateDirectory(dirPath string, createIfNotExist bool) error {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		if !createIfNotExist {
			return fmt.Errorf("directory does not exist at %v", dirPath)
		}
		err = os.MkdirAll(dirPath, 0777)
		if err != nil {
			return fmt.Errorf("failed to create directory at %v: %w", dirPath, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("directory error at %v: %w", dirPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("directory is actually a file at %v", dirPath)
	}
	return nil
}

func splitListFlag(flag string) []string {
	if len(flag) == 0 {
		return []string{}
	}
	return strings.Split(flag, ",")
}
