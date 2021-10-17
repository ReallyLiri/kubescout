package store

import (
	"crypto"
	"fmt"
)

func Hash(items ...interface{}) (string, error) {
	digester := crypto.MD5.New()
	for _, item := range items {
		_, err := fmt.Fprint(digester, item)
		if err != nil {
			return "", fmt.Errorf("failed to calculate hash on item '%v': %v", item, err)
		}
	}
	return fmt.Sprintf("%x", digester.Sum(nil)), nil
}
