package logjson

import "testing"

func TestLogjsonMD5StructTag(t *testing.T) {
	type Msg struct {
		Body    string  `json:"body" logjson:"md5"`
		BodyPtr *string `json:"body_ptr" logjson:"md5"`
		Data    []byte  `json:"data" logjson:"md5"`
	}

	bodyPtr := "pointer"
	msg := Msg{Body: "world", BodyPtr: &bodyPtr, Data: []byte("bytes")}
	got, err := Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"body":"` + md5Expect("world") + `","body_ptr":"` + md5Expect("pointer") + `","data":"` + md5Expect("bytes") + `"}`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
