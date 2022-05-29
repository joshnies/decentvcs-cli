package storage

import (
	"io"

	"github.com/klauspost/compress/zstd"
)

// Compress input to output.
//
// @param in - Input to compress.
//
// @param out - Output to write compressed data to.
//
func Compress(in io.Reader, out io.Writer) error {
	enc, err := zstd.NewWriter(out)
	if err != nil {
		return err
	}

	// Copy content...
	_, err = io.Copy(enc, in)
	if err != nil {
		enc.Close()
		return err
	}
	return enc.Close()
}

// Decompress input to output.
//
// @param in - Input to decompress.
//
// @param out - Output to write decompressed data to.
//
func Decompress(in io.Reader, out io.Writer) error {
	d, err := zstd.NewReader(in)
	if err != nil {
		return err
	}
	defer d.Close()

	// Copy content...
	_, err = io.Copy(out, d)
	return err
}
