package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFloatParam(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		key     string
		want    float64
		wantErr bool
	}{
		{"missing key returns zero, no error", map[string]interface{}{}, "x", 0, false},
		{"float64 value passes through", map[string]interface{}{"x": 3.5}, "x", 3.5, false},
		{"int value is coerced to float64", map[string]interface{}{"x": 7}, "x", 7, false},
		{"string value is an error", map[string]interface{}{"x": "nope"}, "x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := floatParam(tt.params, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIntParam(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		key     string
		want    int
		wantErr bool
	}{
		{"missing key returns zero, no error", map[string]interface{}{}, "n", 0, false},
		{"float64 value truncates to int", map[string]interface{}{"n": 4.9}, "n", 4, false},
		{"int value passes through", map[string]interface{}{"n": 12}, "n", 12, false},
		{"string value is an error", map[string]interface{}{"n": "nope"}, "n", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := intParam(tt.params, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringParam(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		key     string
		want    string
		wantErr bool
	}{
		{"missing key returns empty string, no error", map[string]interface{}{}, "s", "", false},
		{"string value passes through", map[string]interface{}{"s": "hello"}, "s", "hello", false},
		{"non-string value is an error", map[string]interface{}{"s": 5}, "s", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stringParam(tt.params, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
