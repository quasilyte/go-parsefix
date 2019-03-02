package main

import (
	"bytes"
)

func newSrcFile(data []byte) *srcFile {
	var lines [][]byte

	// The sentinel line.
	lines = append(lines, []byte{'\n'})

	// Note that we want '\n' to present in every line,
	// this is why `bytes.Split` on it's own is not enough.
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = append(line, '\n')
		lines = append(lines, line)
	}

	return &srcFile{lines: lines}
}

type srcFile struct {
	lines [][]byte
}

func (src *srcFile) Bytes() []byte {
	var out []byte
	// Skip the first, sentinel line.
	for _, line := range src.lines[1:] {
		out = append(out, line...)
	}
	return out
}
