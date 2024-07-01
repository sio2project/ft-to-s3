package utils

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
)

func ReadGzip(reader io.Reader) ([]byte, error) {
	filegz, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer filegz.Close()

	return io.ReadAll(filegz)
}

func Sha256Checksum(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}
