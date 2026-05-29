package obs

import (
	"github.com/ethanvc/evo/logjson"
)

type PreferJson struct {
	v any
}

func PreferJSON(v any) PreferJson {
	return PreferJson{v: v}
}

func preferJSONPayload(v any) (b []byte, ok bool) {
	switch x := v.(type) {
	case string:
		return []byte(x), true
	case []byte:
		return x, true
	case *string:
		if x == nil {
			return nil, false
		}
		return []byte(*x), true
	case *[]byte:
		if x == nil {
			return nil, false
		}
		return *x, true
	default:
		return nil, false
	}
}

func (p *PreferJson) MarshalJSONTo(enc *logjson.Encoder) error {
	b, ok := preferJSONPayload(p.v)
	if !ok {
		return logjson.MarshalEncode(enc, p.v)
	}
	if logjson.Value(b).IsValid() {
		return enc.WriteValue(logjson.Value(b))
	}
	return enc.WriteToken(logjson.TokenString(string(b)))
}

func (p *PreferJson) MarshalJSON() ([]byte, error) {
	b, ok := preferJSONPayload(p.v)
	if !ok {
		return logjson.Marshal(p.v)
	}
	if logjson.Value(b).IsValid() {
		out := make([]byte, len(b))
		copy(out, b)
		return out, nil
	}
	return logjson.Marshal(string(b))
}
