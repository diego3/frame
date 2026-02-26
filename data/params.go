package data

import "fmt"

// floatParam returns a float64 from params (YAML unmarshals numbers as float64).
func floatParam(params map[string]interface{}, key string) (float64, error) {
	v, ok := params[key]
	if !ok {
		return 0, nil
	}
	switch x := v.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	default:
		return 0, fmt.Errorf("param %q: expected number, got %T", key, v)
	}
}

// intParam returns an int from params.
func intParam(params map[string]interface{}, key string) (int, error) {
	f, err := floatParam(params, key)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}

// stringParam returns a string from params.
func stringParam(params map[string]interface{}, key string) (string, error) {
	v, ok := params[key]
	if !ok {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("param %q: expected string, got %T", key, v)
	}
	return s, nil
}
