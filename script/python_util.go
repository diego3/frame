package script

import (
	"github.com/go-python/gpython/py"
)

// goValueToPy converts a Go value to the closest gpython (py.Object) equivalent.
// Used when passing Go-side data (e.g. event payloads) into Python scripts.
func goValueToPy(v interface{}) py.Object {
	if v == nil {
		return py.None
	}
	switch x := v.(type) {
	case string:
		return py.String(x)
	case float64:
		return py.Float(x)
	case float32:
		return py.Float(float64(x))
	case int:
		return py.Float(float64(x))
	case int64:
		return py.Float(float64(x))
	case bool:
		return py.NewBool(x)
	case map[string]interface{}:
		return goMapToPyDict(x)
	default:
		return py.None
	}
}

// goMapToPyDict converts a Go map[string]interface{} to a py.StringDict so
// Python scripts can access payload keys with payload["key"] syntax.
func goMapToPyDict(m map[string]interface{}) py.StringDict {
	d := make(py.StringDict, len(m))
	for k, v := range m {
		d[k] = goValueToPy(v)
	}
	return d
}

// pyValueToGo converts a gpython object to the closest Go equivalent.
// Used when Python scripts pass data back to Go (e.g. via engine.emit payload).
func pyValueToGo(v py.Object) interface{} {
	switch x := v.(type) {
	case py.String:
		return string(x)
	case py.Float:
		return float64(x)
	case py.Int:
		return float64(x) // normalise to float64 for consistency with Lua
	case py.Bool:
		return bool(x)
	case py.StringDict:
		return pyDictToGoMap(x)
	default:
		return nil
	}
}

// pyDictToGoMap converts a py.StringDict (Python dict) to map[string]interface{}.
func pyDictToGoMap(d py.StringDict) map[string]interface{} {
	out := make(map[string]interface{}, len(d))
	for k, v := range d {
		out[k] = pyValueToGo(v)
	}
	return out
}

// pyToFloat64 extracts a float64 from a py.Float or py.Int object.
// Returns a TypeError if the object is neither.
func pyToFloat64(obj py.Object) (float64, error) {
	switch v := obj.(type) {
	case py.Float:
		return float64(v), nil
	case py.Int:
		return float64(v), nil
	default:
		return 0, py.ExceptionNewf(py.TypeError, "expected a number, got %s", obj.Type().Name)
	}
}

// pyToString extracts a string from a py.String object.
// Returns a TypeError if the object is not a string.
func pyToString(obj py.Object) (string, error) {
	s, ok := obj.(py.String)
	if !ok {
		return "", py.ExceptionNewf(py.TypeError, "expected a string, got %s", obj.Type().Name)
	}
	return string(s), nil
}
