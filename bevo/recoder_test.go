package bevo

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// scriptedReader returns a deterministic byte stream. It emits err exactly at
// the end of the stream (potentially together with the final bytes depending
// on the caller's Read size).
type scriptedReader struct {
	data []byte
	i    int
	err  error
}

func (sr *scriptedReader) Read(p []byte) (int, error) {
	if sr.i >= len(sr.data) {
		if sr.err == nil {
			return 0, io.EOF
		}
		return 0, sr.err
	}
	n := copy(p, sr.data[sr.i:])
	sr.i += n
	if sr.i >= len(sr.data) && sr.err != nil {
		return n, sr.err
	}
	return n, nil
}

func readAllWithSizes(t *testing.T, r io.Reader, sizes []int) (out []byte, errs []error) {
	t.Helper()
	for _, sz := range sizes {
		buf := make([]byte, sz)
		n, err := r.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		errs = append(errs, err)
		if err != nil {
			return out, errs
		}
	}
	tmp := make([]byte, 7)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			out = append(out, tmp[:n]...)
		}
		errs = append(errs, err)
		if err != nil {
			return out, errs
		}
	}
}

func assertStreamEqual(t *testing.T, orig, got []byte, origErrs, gotErrs []error) {
	t.Helper()
	if !bytes.Equal(orig, got) {
		t.Fatalf("stream mismatch: want %q, got %q", orig, got)
	}
	if len(origErrs) == 0 || len(gotErrs) == 0 {
		t.Fatalf("expected terminal errors, got orig=%v rec=%v", origErrs, gotErrs)
	}
	if gotErrs[len(gotErrs)-1] != origErrs[len(origErrs)-1] {
		t.Fatalf("terminal error mismatch: want %v, got %v",
			origErrs[len(origErrs)-1], gotErrs[len(gotErrs)-1])
	}
}

func TestNewRecoder_DefaultMaxBytes(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 2048)
	rec := NewRecoder(bytes.NewReader(data), 0)
	got := rec.Bytes()
	if len(got) != len(data) {
		t.Fatalf("default maxBytes want %d recorded bytes, got %d", len(data), len(got))
	}
}

func TestRecoder_Bytes_RecordsUpToMaxBytes(t *testing.T) {
	src := []byte("hello world")
	rec := NewRecoder(bytes.NewReader(src), 5)
	if got := rec.Bytes(); string(got) != "hello" {
		t.Fatalf("Bytes() = %q, want %q", got, "hello")
	}
}

func TestRecoder_String(t *testing.T) {
	rec := NewRecoder(strings.NewReader("abc"), 2)
	if got := rec.String(); got != "" {
		t.Fatalf("String() before Recod = %q, want empty", got)
	}
	rec.Bytes()
	if got := rec.String(); got != "ab" {
		t.Fatalf("String() after Recod = %q, want %q", got, "ab")
	}
}

func TestRecoder_RecodIdempotent(t *testing.T) {
	rec := NewRecoder(bytes.NewReader([]byte("payload")), 10)
	first := rec.Bytes()
	second := rec.Bytes()
	if !bytes.Equal(first, second) {
		t.Fatalf("Recod/Bytes not idempotent: first=%q second=%q", first, second)
	}
	if string(rec.String()) != "payload" {
		t.Fatalf("String() after repeated Recod = %q", rec.String())
	}
}

func TestRecoder_EmptyInput(t *testing.T) {
	rec := NewRecoder(bytes.NewReader(nil), 8)
	if got := rec.Bytes(); len(got) != 0 {
		t.Fatalf("Bytes() = %q, want empty", got)
	}

	buf := make([]byte, 4)
	n, err := rec.Read(buf)
	if n != 0 || err != io.EOF {
		t.Fatalf("Read() = (%d, %v), want (0, EOF)", n, err)
	}
}

func TestRecoder_ReadBeyondRecordedBytes(t *testing.T) {
	src := []byte("hello world")
	rec := NewRecoder(bytes.NewReader(src), 5)

	if got := rec.Bytes(); string(got) != "hello" {
		t.Fatalf("recorded = %q, want %q", got, "hello")
	}

	out, err := io.ReadAll(rec)
	if err != nil {
		t.Fatalf("ReadAll() err = %v", err)
	}
	if string(out) != string(src) {
		t.Fatalf("ReadAll() = %q, want %q", out, src)
	}
}

func TestRecoder_ReadMatchesOriginal(t *testing.T) {
	tests := []struct {
		name     string
		src      []byte
		maxBytes int
		sizes    []int
		terminal error
	}{
		{
			name:     "eof boundary with mixed read sizes",
			src:      []byte("hello world"),
			maxBytes: 5,
			sizes:    []int{1, 2, 20},
			terminal: io.EOF,
		},
		{
			name:     "custom error with single large read",
			src:      []byte("abcdef"),
			maxBytes: 3,
			sizes:    []int{10},
			terminal: errors.New("boom"),
		},
		{
			name:     "small reads",
			src:      []byte("1234567890"),
			maxBytes: 4,
			sizes:    []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			terminal: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := &scriptedReader{data: append([]byte(nil), tt.src...), err: tt.terminal}
			gotOrig, gotOrigErrs := readAllWithSizes(t, orig, tt.sizes)

			rec := NewRecoder(&scriptedReader{data: append([]byte(nil), tt.src...), err: tt.terminal}, tt.maxBytes)
			gotRec, gotRecErrs := readAllWithSizes(t, rec, tt.sizes)
			assertStreamEqual(t, gotOrig, gotRec, gotOrigErrs, gotRecErrs)

			wantRecorded := tt.src
			if len(wantRecorded) > tt.maxBytes {
				wantRecorded = wantRecorded[:tt.maxBytes]
			}
			if !bytes.Equal(rec.Bytes(), wantRecorded) {
				t.Fatalf("recorded = %q, want %q", rec.Bytes(), wantRecorded)
			}
		})
	}
}

func TestRecoder_ReadFromBufferReturnsStoredError(t *testing.T) {
	rec := NewRecoder(bytes.NewReader([]byte("ab")), 10)

	buf := make([]byte, 1)
	n, err := rec.Read(buf)
	if n != 1 || err != io.EOF {
		t.Fatalf("first Read() = (%d, %v), want (1, EOF)", n, err)
	}
	if buf[0] != 'a' {
		t.Fatalf("first byte = %q, want %q", buf[0], 'a')
	}

	n, err = rec.Read(make([]byte, 1))
	if n != 1 || err != nil {
		t.Fatalf("second Read() = (%d, %v), want (1, nil)", n, err)
	}
}

func TestRecoder_ErrorDuringRecord(t *testing.T) {
	wantErr := errors.New("read failed")
	rec := NewRecoder(&scriptedReader{data: []byte("abc"), err: wantErr}, 10)

	if got := rec.Bytes(); string(got) != "abc" {
		t.Fatalf("Bytes() = %q, want %q", got, "abc")
	}

	buf := make([]byte, 8)
	n, err := rec.Read(buf)
	if n != 3 || !errors.Is(err, wantErr) {
		t.Fatalf("Read() = (%d, %v), want (3, %v)", n, err, wantErr)
	}

	n, err = rec.Read(buf)
	if n != 0 || !errors.Is(err, wantErr) {
		t.Fatalf("second Read() = (%d, %v), want (0, %v)", n, err, wantErr)
	}
}

func TestRecoder_ZeroLengthRead(t *testing.T) {
	rec := NewRecoder(bytes.NewReader([]byte("x")), 1)
	if _, err := rec.Read(nil); err != nil {
		t.Fatalf("Read(nil) err = %v", err)
	}
	if got := rec.Bytes(); string(got) != "x" {
		t.Fatalf("Bytes() = %q, want %q", got, "x")
	}
}

func TestRecoder_ReadAllInOneShot(t *testing.T) {
	src := []byte("short")
	rec := NewRecoder(bytes.NewReader(src), 3)

	got, err := io.ReadAll(rec)
	if err != nil {
		t.Fatalf("ReadAll() err = %v", err)
	}
	if !bytes.Equal(got, src) {
		t.Fatalf("ReadAll() = %q, want %q", got, src)
	}
	if !bytes.Equal(rec.Bytes(), src[:3]) {
		t.Fatalf("Bytes() = %q, want %q", rec.Bytes(), src[:3])
	}
}
