package plog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGlobalIgnore(t *testing.T) {
	defer DefaultEncoder().ClearAll()
	type Abc struct {
		Name string
	}
	abc := Abc{Name: "hello"}
	DefaultEncoder().Set("Name", Ignore)
	require.Equal(t, `{"Name":""}`, toJson(abc))
}

func TestGlobalMd5(t *testing.T) {
	defer DefaultEncoder().ClearAll()
	type Abc struct {
		Name string
	}
	abc := Abc{Name: "hello"}
	DefaultEncoder().Set("Name", Md5)
	require.Equal(t, `{"Name":"5d41402abc4b2a76b9719d911017c592(5)"}`, toJson(abc))
}

func TestTagIgnore(t *testing.T) {
	type Abc struct {
		Name string `evolog:"ignore"`
	}
	abc := Abc{Name: "hello"}
	require.Equal(t, `{"Name":""}`, toJson(abc))
}

func TestTagMd5(t *testing.T) {
	type Abc struct {
		Name string `evolog:"md5"`
	}
	abc := Abc{Name: "hello"}
	require.Equal(t, `{"Name":"5d41402abc4b2a76b9719d911017c592(5)"}`, toJson(abc))
}

func toJson(v any) string {
	buf, _ := Marshal(v)
	return string(buf)
}
