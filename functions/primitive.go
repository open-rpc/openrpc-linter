package functions

import "strconv"

// primitiveKey keeps type identity separate from its normalized value.
// Display formatting is needed only when a duplicate produces a diagnostic.
type primitiveKind uint8

const (
	primitiveNull primitiveKind = iota
	primitiveString
	primitiveBool
	primitiveNumber
)

type primitiveKey struct {
	kind  primitiveKind
	value string
}

func comparableKey(value any) (primitiveKey, bool) {
	switch value := value.(type) {
	case nil:
		return primitiveKey{kind: primitiveNull}, true
	case string:
		return primitiveKey{kind: primitiveString, value: value}, true
	case bool:
		return primitiveKey{kind: primitiveBool, value: strconv.FormatBool(value)}, true
	case float64:
		return primitiveKey{kind: primitiveNumber, value: strconv.FormatFloat(value, 'g', -1, 64)}, true
	default:
		return primitiveKey{}, false
	}
}

func (key primitiveKey) display() string {
	switch key.kind {
	case primitiveNull:
		return "null"
	case primitiveString:
		return strconv.Quote(key.value)
	default:
		return key.value
	}
}
