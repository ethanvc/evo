package logjson

import (
	"io"

	"github.com/ethanvc/evo/logjson/internal/gostd/encoding/json/jsontext"
	jsonv2 "github.com/ethanvc/evo/logjson/internal/gostd/encoding/json/v2"
)

type (
	Encoder = jsontext.Encoder
	Token   = jsontext.Token
	Kind    = jsontext.Kind
	Value   = jsontext.Value
	Options = jsontext.Options
)

var (
	Null  = jsontext.Null
	False = jsontext.False
	True  = jsontext.True

	BeginObject = jsontext.BeginObject
	EndObject   = jsontext.EndObject
	BeginArray  = jsontext.BeginArray
	EndArray    = jsontext.EndArray
)

var (
	TokenString = jsontext.String
	TokenInt    = jsontext.Int
	TokenUint   = jsontext.Uint
	TokenFloat  = jsontext.Float
	TokenBool   = jsontext.Bool
)

var NewEncoder = jsontext.NewEncoder

func AllowDuplicateNames(v bool) Options { return jsontext.AllowDuplicateNames(v) }

func NewEncoderOf(w io.Writer) *Encoder {
	return NewEncoder(w, AllowDuplicateNames(true))
}

var Marshal = jsonv2.Marshal
var MarshalEncode = jsonv2.MarshalEncode
