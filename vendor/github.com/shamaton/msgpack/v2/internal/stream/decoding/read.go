package decoding

import (
	"fmt"
	"io"
)

// readFull fills b completely from d.r. The common case, where the
// underlying reader already returns the full amount requested, completes
// with a single Read call. Readers that legitimately return fewer bytes than
// requested (n > 0, err == nil) fall back to looping over the remainder.
// Unlike io.ReadFull, an EOF reached only after some bytes were already
// consumed by this call is still reported as io.EOF, not io.ErrUnexpectedEOF,
// preserving this package's historical error values.
func (d *decoder) readFull(b []byte) error {
	for len(b) > 0 {
		n, err := d.r.Read(b)
		if n < 0 || n > len(b) {
			return io.ErrNoProgress
		}
		b = b[n:]
		if len(b) == 0 {
			return nil
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func (d *decoder) readSize1() (byte, error) {
	if err := d.readFull(d.buf.B1); err != nil {
		return 0, err
	}
	return d.buf.B1[0], nil
}

func (d *decoder) readSize2() ([]byte, error) {
	if err := d.readFull(d.buf.B2); err != nil {
		return emptyBytes, err
	}
	return d.buf.B2, nil
}

func (d *decoder) readSize4() ([]byte, error) {
	if err := d.readFull(d.buf.B4); err != nil {
		return emptyBytes, err
	}
	return d.buf.B4, nil
}

func (d *decoder) readSize8() ([]byte, error) {
	if err := d.readFull(d.buf.B8); err != nil {
		return emptyBytes, err
	}
	return d.buf.B8, nil
}

func (d *decoder) readSize16() ([]byte, error) {
	if err := d.readFull(d.buf.B16); err != nil {
		return emptyBytes, err
	}
	return d.buf.B16, nil
}

func (d *decoder) readSizeN(n int) ([]byte, error) {
	if n < 0 {
		return emptyBytes, fmt.Errorf("invalid declared byte length %d", n)
	}
	if n <= len(d.buf.Data) {
		b := d.buf.Data[:n]
		if err := d.readFull(b); err != nil {
			return emptyBytes, err
		}
		return b, nil
	}
	if n <= maxPreallocBytes {
		b := make([]byte, n)
		if err := d.readFull(b); err != nil {
			return emptyBytes, err
		}
		return b, nil
	}
	return d.readSizeNGrowing(n)
}

// readSizeNGrowing reads a declared byte length that exceeds maxPreallocBytes.
// Rather than accumulating fixed-size chunks with append (which pays for a
// separate chunk buffer plus a copy into the growing slice on every
// iteration), it doubles the output slice's own capacity and reads directly
// into the newly available space, halving the bytes copied for large,
// legitimate payloads.
func (d *decoder) readSizeNGrowing(n int) ([]byte, error) {
	b := make([]byte, 0, initialByteCap(n))
	for len(b) < n {
		if len(b) == cap(b) {
			newCap := n
			if remaining := n - cap(b); remaining > cap(b) {
				newCap = cap(b) * 2
			}
			grown := make([]byte, len(b), newCap)
			copy(grown, b)
			b = grown
		}

		start := len(b)
		end := cap(b)
		if n < end {
			end = n
		}
		b = b[:end]
		if err := d.readFull(b[start:end]); err != nil {
			return emptyBytes, err
		}
	}
	return b, nil
}
