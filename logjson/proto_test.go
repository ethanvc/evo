package logjson

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"testing"
)

func md5Expect(s string) string {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("len=%d,%s", len(s), hex.EncodeToString(h[:]))
}

func TestProtoLogjsonMD5(t *testing.T) {
	msg := &ProtoTest{Secret: "hello"}
	got, err := Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"secret":"` + md5Expect("hello") + `"}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestProtoLogjsonMD5Empty(t *testing.T) {
	msg := &ProtoTest{Secret: ""}
	got, err := Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"secret":"` + md5Expect("") + `"}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestNonProtoStructUnaffected(t *testing.T) {
	type Plain struct {
		Name string `json:"name"`
	}
	got, err := Marshal(Plain{Name: "world"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"world"}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
