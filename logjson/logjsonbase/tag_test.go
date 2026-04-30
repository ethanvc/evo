package logjsonbase

import "testing"

func TestParseTag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want TagOptions
	}{
		{
			name: "empty",
			tag:  "",
			want: TagOptions{},
		},
		{
			name: "md5",
			tag:  "md5",
			want: TagOptions{MD5: true},
		},
		{
			name: "trimmed md5",
			tag:  " unknown, md5 ",
			want: TagOptions{MD5: true},
		},
		{
			name: "unknown",
			tag:  "unknown",
			want: TagOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTag(tt.tag)
			if got != tt.want {
				t.Fatalf("ParseTag(%q) = %+v, want %+v", tt.tag, got, tt.want)
			}
		})
	}
}
