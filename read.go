package goramwordnet

import (
	"bufio"
	"io"
	"os"
)

// InPlaceReadLine scans a file and invoke the provided callback for
// every line read.  Because scanning a file for newline delimiters is
// an incredibly cheap operation, the overhead of multi-thread
// communication can be slower than processing on a single thread when
// the per line computational cost is low.
func inPlaceReadLine(s io.Reader, cb func([]byte, int64, int64) error) error {
	const bufSize = 8396800 // 8 meg
	reader := bufio.NewReaderSize(s, bufSize)
	count := int64(1)
	var offset int64
	var err error
	var line []byte
	for line, err = reader.ReadSlice('\n'); err == nil; line, err = reader.ReadSlice('\n') {
		if err = cb(line[:len(line)-1], count, offset); err != nil {
			return err
		}
		offset += int64(len(line))
		count++
	}
	// If we reached end of file and the line contents are empty, don't return an additional line.
	if err == io.EOF {
		err = nil
		if len(line) > 0 {
			return cb(line, count, offset)
		}
	} else {
		return cb(line, count, offset)
	}
	return nil
}

func inPlaceReadLineFromPath(filePath string, cb func([]byte, int64, int64) error) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return inPlaceReadLine(f, cb)
}
