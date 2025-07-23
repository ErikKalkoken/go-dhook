package dhook

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLength(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"alpha 😀 boy", 11},
		{"alpha boy", 9},
		{"", 0},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("#%d", i+1), func(t *testing.T) {
			got := length(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsValidPublicURL(t *testing.T) {
	cases := []struct {
		name, rawURL string
		want         bool
		hasError     bool
	}{
		{"all good", "https://www.googl.com", true, false},
		{"invalid URL", "//invalid/server/abc", false, false},
		{"URL parse error", "xxx", false, true},
		{"empty URLs are valid", "", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := isValidPublicURL(tc.rawURL)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, tc.want, got)
				}
			}
		})
	}
}
