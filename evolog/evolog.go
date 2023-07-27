package evolog

import "github.com/ethanvc/evo/evojson"

func Marshal(v any, encoder ...*Encoder) ([]byte, error) {
	return evojson.Marshal(v, getEncoder(encoder...).configer.Load())
}

func getEncoder(encoder ...*Encoder) *Encoder {
	for _, enc := range encoder {
		if enc != nil {
			return enc
		}
	}
	return DefaultEncoder()
}
