package bevo

import (
	"errors"
	"io"
	"testing"
)

// scriptedReader returns a deterministic byte stream. It emits err exactly at
// the end of the stream (potentially together with the final bytes depending
// on the caller's Read size), which is sufficient to validate error-boundary
// equivalence for Recoder.
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
	// Drain to termination so we always capture the terminal error.
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

func TestRecoder_RecodDoesNotChangeStream_EOFBoundary(t *testing.T) {
	src := []byte("hello world")
	sizes := []int{1, 2, 20} // includes a read that spans EOF boundary

	orig := &scriptedReader{data: append([]byte(nil), src...), err: io.EOF}
	gotOrig, gotOrigErrs := readAllWithSizes(t, orig, sizes)

	rec := NewRecoder(&scriptedReader{data: append([]byte(nil), src...), err: io.EOF}, 5)
	rec.Recod()
	gotRec, gotRecErrs := readAllWithSizes(t, rec, sizes)

	if string(gotRec) != string(gotOrig) {
		t.Fatalf("stream mismatch: want %q, got %q", string(gotOrig), string(gotRec))
	}
	if len(gotRecErrs) == 0 || len(gotOrigErrs) == 0 {
		t.Fatalf("expected terminal errors, got orig=%v rec=%v", gotOrigErrs, gotRecErrs)
	}
	if gotRecErrs[len(gotRecErrs)-1] != gotOrigErrs[len(gotOrigErrs)-1] {
		t.Fatalf("terminal error mismatch: want %v, got %v", gotOrigErrs[len(gotOrigErrs)-1], gotRecErrs[len(gotRecErrs)-1])
	}

	if string(rec.Bytes()) != "hello" {
		t.Fatalf("recorded bytes mismatch: want %q, got %q", "hello", string(rec.Bytes()))
	}
}

func TestRecoder_RecodDoesNotChangeStream_CustomError(t *testing.T) {
	src := []byte("abcdef")
	wantErr := errors.New("boom")
	sizes := []int{10} // single read spans boundary and should return err with bytes

	orig := &scriptedReader{data: append([]byte(nil), src...), err: wantErr}
	gotOrig, gotOrigErrs := readAllWithSizes(t, orig, sizes)

	rec := NewRecoder(&scriptedReader{data: append([]byte(nil), src...), err: wantErr}, 3)
	rec.Recod()
	gotRec, gotRecErrs := readAllWithSizes(t, rec, sizes)

	if string(gotRec) != string(gotOrig) {
		t.Fatalf("stream mismatch: want %q, got %q", string(gotOrig), string(gotRec))
	}
	if gotRecErrs[len(gotRecErrs)-1] != wantErr {
		t.Fatalf("terminal error mismatch: want %v, got %v", wantErr, gotRecErrs[len(gotRecErrs)-1])
	}
	if gotOrigErrs[len(gotOrigErrs)-1] != wantErr {
		t.Fatalf("orig terminal error mismatch: want %v, got %v", wantErr, gotOrigErrs[len(gotOrigErrs)-1])
	}
	if string(rec.Bytes()) != "abc" {
		t.Fatalf("recorded bytes mismatch: want %q, got %q", "abc", string(rec.Bytes()))
	}
}

