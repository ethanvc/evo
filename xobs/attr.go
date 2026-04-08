package xobs

import (
	"fmt"
	"math"
	"strconv"
	"time"
	"unsafe"
)

// Attr is a key-value pair for structured logging.
type Attr struct {
	Key string
	Val Value
}

func String(key, val string) Attr   { return Attr{Key: key, Val: StringValue(val)} }
func Int64(key string, val int64) Attr  { return Attr{Key: key, Val: Int64Value(val)} }
func Int(key string, val int) Attr    { return Attr{Key: key, Val: Int64Value(int64(val))} }
func Uint64(key string, val uint64) Attr { return Attr{Key: key, Val: Uint64Value(val)} }
func Float64(key string, val float64) Attr { return Attr{Key: key, Val: Float64Value(val)} }
func Bool(key string, val bool) Attr   { return Attr{Key: key, Val: BoolValue(val)} }
func Time(key string, val time.Time) Attr { return Attr{Key: key, Val: TimeValue(val)} }
func Duration(key string, val time.Duration) Attr { return Attr{Key: key, Val: DurationValue(val)} }
func Any(key string, val any) Attr { return Attr{Key: key, Val: AnyValue(val)} }

func (a Attr) Equal(b Attr) bool {
	return a.Key == b.Key && a.Val.Equal(b.Val)
}

// Value can represent any Go value, but unlike type any,
// it can represent most small values without an allocation.
// The zero Value corresponds to nil.
type Value struct {
	_ [0]func() // disallow ==
	// num holds the value for Int64, Uint64, Float64, Bool and Duration,
	// the string length for KindString, and nanoseconds since the epoch for KindTime.
	num uint64
	// If any is of type Kind, then the value is in num.
	// If any is of type *time.Location, then Kind is Time.
	// If any is of type stringptr, then Kind is String.
	// Otherwise, Kind is Any and any is the value.
	any any
}

type (
	stringptr    *byte
	timeLocation *time.Location
	timeTime     time.Time
)

// Kind is the type of a Value.
type Kind int

const (
	KindAny Kind = iota
	KindBool
	KindDuration
	KindFloat64
	KindInt64
	KindString
	KindTime
	KindUint64
	KindLogValuer
)

var kindStrings = []string{
	"Any", "Bool", "Duration", "Float64", "Int64",
	"String", "Time", "Uint64", "LogValuer",
}

func (k Kind) String() string {
	if k >= 0 && int(k) < len(kindStrings) {
		return kindStrings[k]
	}
	return "<unknown Kind>"
}

// Unexported wrapper so user-provided Kind values fall through to KindAny.
type kind Kind

func (v Value) Kind() Kind {
	switch x := v.any.(type) {
	case Kind:
		return x
	case stringptr:
		return KindString
	case timeLocation, timeTime:
		return KindTime
	case LogValuer:
		return KindLogValuer
	case kind:
		_ = x
		return KindAny
	default:
		return KindAny
	}
}

//////////////// Constructors

func StringValue(val string) Value {
	return Value{num: uint64(len(val)), any: stringptr(unsafe.StringData(val))}
}

func Int64Value(val int64) Value {
	return Value{num: uint64(val), any: KindInt64}
}

func IntValue(val int) Value {
	return Int64Value(int64(val))
}

func Uint64Value(val uint64) Value {
	return Value{num: val, any: KindUint64}
}

func Float64Value(val float64) Value {
	return Value{num: math.Float64bits(val), any: KindFloat64}
}

func BoolValue(val bool) Value {
	u := uint64(0)
	if val {
		u = 1
	}
	return Value{num: u, any: KindBool}
}

func TimeValue(val time.Time) Value {
	if val.IsZero() {
		return Value{any: timeLocation(nil)}
	}
	nsec := val.UnixNano()
	t := time.Unix(0, nsec)
	if val.Equal(t) {
		return Value{num: uint64(nsec), any: timeLocation(val.Location())}
	}
	return Value{any: timeTime(val.Round(0))}
}

func DurationValue(val time.Duration) Value {
	return Value{num: uint64(val.Nanoseconds()), any: KindDuration}
}

// AnyValue returns a Value for the supplied value,
// using the most specific Kind when possible.
func AnyValue(val any) Value {
	switch v := val.(type) {
	case string:
		return StringValue(v)
	case int:
		return Int64Value(int64(v))
	case int64:
		return Int64Value(v)
	case uint64:
		return Uint64Value(v)
	case bool:
		return BoolValue(v)
	case float64:
		return Float64Value(v)
	case float32:
		return Float64Value(float64(v))
	case time.Duration:
		return DurationValue(v)
	case time.Time:
		return TimeValue(v)
	case uint:
		return Uint64Value(uint64(v))
	case int8:
		return Int64Value(int64(v))
	case int16:
		return Int64Value(int64(v))
	case int32:
		return Int64Value(int64(v))
	case uint8:
		return Uint64Value(uint64(v))
	case uint16:
		return Uint64Value(uint64(v))
	case uint32:
		return Uint64Value(uint64(v))
	case uintptr:
		return Uint64Value(uint64(v))
	case Kind:
		return Value{any: kind(v)}
	case Value:
		return v
	default:
		return Value{any: v}
	}
}

//////////////// Accessors

func (v Value) Any() any {
	switch v.Kind() {
	case KindAny:
		if k, ok := v.any.(kind); ok {
			return Kind(k)
		}
		return v.any
	case KindLogValuer:
		return v.any
	case KindInt64:
		return int64(v.num)
	case KindUint64:
		return v.num
	case KindFloat64:
		return v.float64()
	case KindString:
		return v.str()
	case KindBool:
		return v.bool()
	case KindDuration:
		return v.duration()
	case KindTime:
		return v.time()
	default:
		panic(fmt.Sprintf("bad Kind: %s", v.Kind()))
	}
}

func (v Value) String() string {
	if sp, ok := v.any.(stringptr); ok {
		return unsafe.String(sp, v.num)
	}
	var buf []byte
	return string(v.append(buf))
}

func (v Value) str() string {
	return unsafe.String(v.any.(stringptr), v.num)
}

func (v Value) Int64() int64 {
	if g, w := v.Kind(), KindInt64; g != w {
		panic(fmt.Sprintf("Value Kind is %s, not %s", g, w))
	}
	return int64(v.num)
}

func (v Value) Uint64() uint64 {
	if g, w := v.Kind(), KindUint64; g != w {
		panic(fmt.Sprintf("Value Kind is %s, not %s", g, w))
	}
	return v.num
}

func (v Value) Float64() float64 {
	if g, w := v.Kind(), KindFloat64; g != w {
		panic(fmt.Sprintf("Value Kind is %s, not %s", g, w))
	}
	return v.float64()
}

func (v Value) float64() float64 {
	return math.Float64frombits(v.num)
}

func (v Value) Bool() bool {
	if g, w := v.Kind(), KindBool; g != w {
		panic(fmt.Sprintf("Value Kind is %s, not %s", g, w))
	}
	return v.bool()
}

func (v Value) bool() bool {
	return v.num == 1
}

func (v Value) Duration() time.Duration {
	if g, w := v.Kind(), KindDuration; g != w {
		panic(fmt.Sprintf("Value Kind is %s, not %s", g, w))
	}
	return v.duration()
}

func (v Value) duration() time.Duration {
	return time.Duration(int64(v.num))
}

func (v Value) Time() time.Time {
	if g, w := v.Kind(), KindTime; g != w {
		panic(fmt.Sprintf("Value Kind is %s, not %s", g, w))
	}
	return v.time()
}

func (v Value) time() time.Time {
	switch a := v.any.(type) {
	case timeLocation:
		if a == nil {
			return time.Time{}
		}
		return time.Unix(0, int64(v.num)).In((*time.Location)(a))
	case timeTime:
		return time.Time(a)
	default:
		panic(fmt.Sprintf("bad time type %T", v.any))
	}
}

func (v Value) LogValuer() LogValuer {
	return v.any.(LogValuer)
}

//////////////// Other

func (v Value) Equal(w Value) bool {
	k1, k2 := v.Kind(), w.Kind()
	if k1 != k2 {
		return false
	}
	switch k1 {
	case KindInt64, KindUint64, KindBool, KindDuration:
		return v.num == w.num
	case KindString:
		return v.str() == w.str()
	case KindFloat64:
		return v.float64() == w.float64()
	case KindTime:
		return v.time().Equal(w.time())
	case KindAny, KindLogValuer:
		return v.any == w.any
	default:
		panic(fmt.Sprintf("bad Kind: %s", k1))
	}
}

func (v Value) append(dst []byte) []byte {
	switch v.Kind() {
	case KindString:
		return append(dst, v.str()...)
	case KindInt64:
		return strconv.AppendInt(dst, int64(v.num), 10)
	case KindUint64:
		return strconv.AppendUint(dst, v.num, 10)
	case KindFloat64:
		return strconv.AppendFloat(dst, v.float64(), 'g', -1, 64)
	case KindBool:
		return strconv.AppendBool(dst, v.bool())
	case KindDuration:
		return append(dst, v.duration().String()...)
	case KindTime:
		return v.time().AppendFormat(dst, time.RFC3339Nano)
	case KindAny, KindLogValuer:
		return fmt.Append(dst, v.any)
	default:
		panic(fmt.Sprintf("bad Kind: %s", v.Kind()))
	}
}

// LogValuer is any Go value that can convert itself into a Value for logging.
// This mechanism may be used to defer expensive operations until they are needed.
type LogValuer interface {
	LogValue() Value
}

const maxLogValues = 100

// Resolve repeatedly calls LogValue on v while it implements LogValuer,
// and returns the result.
func (v Value) Resolve() Value {
	for i := 0; i < maxLogValues; i++ {
		if v.Kind() != KindLogValuer {
			return v
		}
		v = v.LogValuer().LogValue()
	}
	return AnyValue(fmt.Errorf("LogValue called too many times on Value of type %T", v.Any()))
}
