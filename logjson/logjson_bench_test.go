//go:build goexperiment.jsonv2

package logjson

import (
	stdjsonv2 "encoding/json/v2"
	"testing"
)

type benchmarkMD5Msg struct {
	Body    string  `json:"body" logjson:"md5"`
	BodyPtr *string `json:"body_ptr" logjson:"md5"`
	Data    []byte  `json:"data" logjson:"md5"`
}

var benchmarkMarshalSink []byte

func BenchmarkMarshalMD5ComparedToStdJSONV2(b *testing.B) {
	msg := benchmarkMD5Msg{
		Body:    "world",
		BodyPtr: new("pointer"),
		Data:    []byte("bytes"),
	}

	b.Run("std_jsonv2", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			got, err := stdjsonv2.Marshal(msg)
			if err != nil {
				b.Fatal(err)
			}
			benchmarkMarshalSink = got
		}
	})

	b.Run("logjson", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			got, err := Marshal(msg)
			if err != nil {
				b.Fatal(err)
			}
			benchmarkMarshalSink = got
		}
	})
}
