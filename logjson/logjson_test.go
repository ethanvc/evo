package logjson

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogjsonMD5StructTag(t *testing.T) {
	type Msg struct {
		Body    string  `json:"body" logjson:"md5"`
		BodyPtr *string `json:"body_ptr" logjson:"md5"`
		Data    []byte  `json:"data" logjson:"md5"`
	}

	msg := Msg{Body: "world", BodyPtr: new("pointer"), Data: []byte("bytes")}
	got, err := Marshal(msg)
	require.NoError(t, err)
	require.Equal(t, `{"body":"len=5,7d793037a0760186574b0282f2f435e7","body_ptr":"len=7,ccac8a66d468e2522611be86933cc0d9","data":"len=5,4b3a6218bb3e3a7303e8a171a60fcf92"}`, string(got))
}
