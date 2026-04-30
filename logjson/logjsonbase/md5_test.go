package logjsonbase

import "testing"

func TestLogMd5(t *testing.T) {
	tests := []struct {
		name string
		msg  []byte
		want string
	}{
		{
			name: "empty",
			msg:  nil,
			want: "len=0,d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name: "hello",
			msg:  []byte("hello"),
			want: "len=5,5d41402abc4b2a76b9719d911017c592",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LogMd5(tt.msg); got != tt.want {
				t.Fatalf("LogMd5(%q) = %q, want %q", tt.msg, got, tt.want)
			}
		})
	}
}
