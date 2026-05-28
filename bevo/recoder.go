package bevo

import (
	"io"
)

type Recoder struct {
	reader   io.Reader
	maxBytes int
	buf      []byte
	off      int
	finished bool
	err      error
}

func NewRecoder(w io.Reader, maxBytes int) *Recoder {
	if maxBytes == 0 {
		maxBytes = 1024 * 1024
	}
	return &Recoder{
		reader:   w,
		maxBytes: maxBytes,
	}
}

func (r *Recoder) Recod() {
	if r.finished {
		return
	}
	for len(r.buf) < r.maxBytes {
		oldLen := len(r.buf)
		if oldLen >= r.maxBytes {
			break
		}
		s := min(r.maxBytes-oldLen, 1024)
		r.buf = append(r.buf, make([]byte, s)...)
		n, err := r.reader.Read(r.buf[oldLen:])
		r.buf = r.buf[:oldLen+n]
		if err != nil {
			r.err = err
			break
		}
	}
	r.finished = true
}

func (r *Recoder) Read(p []byte) (n int, err error) {
	r.Recod()
	if r.off < len(r.buf) {
		n = min(len(p), len(r.buf)-r.off)
		copy(p, r.buf[r.off:r.off+n])
		r.off += n
		err = r.err
		r.err = nil
		return n, err
	}
	if r.err != nil {
		err = r.err
		r.err = nil
		return 0, err
	}
	return r.reader.Read(p)
}

func (r *Recoder) Bytes() []byte {
	r.Recod()
	return r.buf
}

func (r *Recoder) String() string {
	return string(r.buf)
}

func (r *Recoder) Close() error {
	if readerCloser, ok := r.reader.(io.ReadCloser); ok {
		return readerCloser.Close()
	}
	return nil
}
