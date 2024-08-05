package storage

import "io"

type GetResult struct {
	Found        bool
	File         io.Reader
	Gziped       bool
	LastModified int64
	LogicalSize  int64
	Err          error
}
