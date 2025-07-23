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

func TestURLValidation(t *testing.T) {
	validURL := "https://www.googl.com"
	invalidURL := "//invalid/server/abc"
	t.Run("author URL", func(t *testing.T) {
		x1 := Embed{Author: EmbedAuthor{URL: validURL}}
		assert.NoError(t, x1.validate())
		x2 := Embed{Author: EmbedAuthor{URL: invalidURL}}
		assert.ErrorIs(t, x2.validate(), ErrInvalidMessage)
	})
	t.Run("author icon URL", func(t *testing.T) {
		x1 := Embed{Author: EmbedAuthor{IconURL: validURL}}
		assert.NoError(t, x1.validate())
		x2 := Embed{Author: EmbedAuthor{IconURL: invalidURL}}
		assert.ErrorIs(t, x2.validate(), ErrInvalidMessage)
	})
	t.Run("image URL", func(t *testing.T) {
		x1 := Embed{Image: EmbedImage{URL: validURL}}
		assert.NoError(t, x1.validate())
		x2 := Embed{Image: EmbedImage{URL: invalidURL}}
		assert.ErrorIs(t, x2.validate(), ErrInvalidMessage)
	})
	t.Run("footer icon URL", func(t *testing.T) {
		x1 := Embed{Footer: EmbedFooter{IconURL: validURL}}
		assert.NoError(t, x1.validate())
		x2 := Embed{Footer: EmbedFooter{IconURL: invalidURL}}
		assert.ErrorIs(t, x2.validate(), ErrInvalidMessage)
	})
	t.Run("thumbnail URL", func(t *testing.T) {
		x1 := Embed{Thumbnail: EmbedThumbnail{URL: validURL}}
		assert.NoError(t, x1.validate())
		x2 := Embed{Thumbnail: EmbedThumbnail{URL: invalidURL}}
		assert.ErrorIs(t, x2.validate(), ErrInvalidMessage)
	})
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
