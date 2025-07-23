package dhook_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/go-dhook"
)

func TestMessage_Validate(t *testing.T) {
	validURL := "https://www.googl.com"
	invalidURL := "//invalid/server/abc"
	cases := []struct {
		name string
		m    dhook.Message
		ok   bool
	}{
		// messages
		{"minimal", dhook.Message{Content: "content"}, true},
		{"empty", dhook.Message{}, false},
		{"content too long", dhook.Message{Content: makeStr(2001)}, false},

		// embed
		{"minimal embed", dhook.Message{Embeds: []dhook.Embed{{Description: "description"}}}, true},
		{
			"embed too large",
			dhook.Message{Embeds: []dhook.Embed{{Description: makeStr(4097)}}},
			false,
		},
		{
			"embed title large",
			dhook.Message{Embeds: []dhook.Embed{{Title: makeStr(257)}}},
			false,
		},
		{
			"combined embeds too large",
			dhook.Message{Embeds: []dhook.Embed{
				{Description: makeStr(4096)},
				{Description: makeStr(4096)},
			}},
			false,
		},
		{
			"username too long",
			dhook.Message{Content: "content", Username: makeStr(81)},
			false,
		},
		{
			"too many embeds",
			dhook.Message{Embeds: []dhook.Embed{
				{Description: "description"},
				{Description: "description"},
				{Description: "description"},
				{Description: "description"},
				{Description: "description"},
				{Description: "description"},
				{Description: "description"},
				{Description: "description"},
				{Description: "description"},
				{Description: "description"},
				{Description: "description"},
			}},
			false,
		},
		{
			"embed with too many fields",
			dhook.Message{Embeds: []dhook.Embed{{Fields: []dhook.Field{
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
				{Name: "name", Value: "value"},
			}}}},
			false,
		},
		// embed field
		{
			"valid field",
			dhook.Message{Embeds: []dhook.Embed{{Fields: []dhook.Field{
				{Name: "name", Value: "value"},
			}}}},
			true,
		},
		{
			"field name too long",
			dhook.Message{Embeds: []dhook.Embed{{Fields: []dhook.Field{
				{Name: makeStr(257), Value: "value"},
			}}}},
			false,
		},
		{
			"field value too long",
			dhook.Message{Embeds: []dhook.Embed{{Fields: []dhook.Field{
				{Name: "name", Value: makeStr(1025)},
			}}}},
			false,
		},
		{
			"field name missing",
			dhook.Message{Embeds: []dhook.Embed{{Fields: []dhook.Field{
				{Name: "", Value: "value"},
			}}}},
			false,
		},
		// embed author
		{
			"valid embed author",
			dhook.Message{Embeds: []dhook.Embed{{Author: dhook.Author{
				Name:    "name",
				URL:     validURL,
				IconURL: validURL,
			}}}},
			true,
		},
		{
			"embed author name too long",
			dhook.Message{Embeds: []dhook.Embed{{Author: dhook.Author{
				Name: makeStr(4096),
			}}}},
			false,
		},
		{
			"invalid embed author URL",
			dhook.Message{Embeds: []dhook.Embed{{Author: dhook.Author{
				Name: "name",
				URL:  invalidURL,
			}}}},
			false,
		},
		{
			"invalid embed author icon URL",
			dhook.Message{Embeds: []dhook.Embed{{Author: dhook.Author{
				Name:    "name",
				IconURL: invalidURL,
			}}}},
			false,
		},
		// embed image
		{
			"valid embed image",
			dhook.Message{Embeds: []dhook.Embed{{Image: dhook.Image{
				URL: validURL,
			}}}},
			true,
		},
		{
			"invalid embed image URL",
			dhook.Message{Embeds: []dhook.Embed{{Image: dhook.Image{
				URL: invalidURL,
			}}}},
			false,
		},
		// embed footer
		{
			"valid embed footer",
			dhook.Message{Embeds: []dhook.Embed{{Footer: dhook.Footer{
				Text:    "Text",
				IconURL: validURL,
			}}}},
			true,
		},
		{
			"embed footer too long",
			dhook.Message{Embeds: []dhook.Embed{{Footer: dhook.Footer{
				Text: makeStr(2049),
			}}}},
			false,
		},
		{
			"invalid icon URL",
			dhook.Message{Embeds: []dhook.Embed{{Footer: dhook.Footer{
				Text:    "Text",
				IconURL: invalidURL,
			}}}},
			false,
		},
		// embed thumbnail
		{
			"valid embed image",
			dhook.Message{Embeds: []dhook.Embed{{Thumbnail: dhook.Image{
				URL: validURL,
			}}}},
			true,
		},
		{
			"invalid embed image URL",
			dhook.Message{Embeds: []dhook.Embed{{Thumbnail: dhook.Image{
				URL: invalidURL,
			}}}},
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.m.Validate()
			if tc.ok {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func makeStr(n int) string {
	return strings.Repeat("x", n)
}
