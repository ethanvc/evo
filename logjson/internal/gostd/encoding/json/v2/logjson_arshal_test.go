package json_test

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"testing"

	json "github.com/ethanvc/evo/logjson/internal/gostd/encoding/json/v2"
)

func md5Val(s string) string {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("len=%d,%s", len(s), hex.EncodeToString(h[:]))
}

func TestLogjsonMD5String(t *testing.T) {
	type Msg struct {
		Name string `json:"name"`
		Body string `json:"body" logjson:"md5"`
	}
	m := Msg{Name: "hello", Body: "world"}
	got, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"hello","body":"` + md5Val("world") + `"}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLogjsonMD5Bytes(t *testing.T) {
	type Msg struct {
		Name string `json:"name"`
		Data []byte `json:"data" logjson:"md5"`
	}
	m := Msg{Name: "hello", Data: []byte("world")}
	got, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"hello","data":"` + md5Val("world") + `"}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLogjsonMD5EmptyString(t *testing.T) {
	type Msg struct {
		Body string `json:"body" logjson:"md5"`
	}
	m := Msg{Body: ""}
	got, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"body":"` + md5Val("") + `"}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLogjsonMD5NilBytes(t *testing.T) {
	type Msg struct {
		Data []byte `json:"data" logjson:"md5"`
	}
	m := Msg{Data: nil}
	got, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"data":"` + md5Val("") + `"}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestLogjsonWithoutMD5(t *testing.T) {
	type Msg struct {
		Name string `json:"name"`
		Body string `json:"body"`
	}
	m := Msg{Name: "hello", Body: "world"}
	got, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"hello","body":"world"}`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
