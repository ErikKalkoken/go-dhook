package dhook_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/go-dhook"
)

func TestMessage_Validate(t *testing.T) {
	cases := []struct {
		name string
		m    dhook.Message
		ok   bool
	}{
		// valid messages
		{"minimal", dhook.Message{Content: "content"}, true},

		// valid embeds
		{"minimal embed", dhook.Message{Embeds: []dhook.Embed{{Description: "description"}}}, true},

		// invalid messages
		{"empty", dhook.Message{}, false},
		{"content too long", dhook.Message{Content: makeStr(2001)}, false},

		// invalid embeds
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
			dhook.Message{Embeds: []dhook.Embed{
				{
					Description: "description",
					Fields: []dhook.EmbedField{
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
					},
				},
			}},
			false,
		},
		// invalid embed fields
		{
			"field name too long",
			dhook.Message{Embeds: []dhook.Embed{
				{
					Description: "description",
					Fields: []dhook.EmbedField{
						{Name: makeStr(257), Value: "value"},
					},
				},
			}},
			false,
		},
		{
			"field value too long",
			dhook.Message{Embeds: []dhook.Embed{
				{
					Description: "description",
					Fields: []dhook.EmbedField{
						{Name: "name", Value: makeStr(1025)},
					},
				},
			}},
			false,
		},
		{
			"field name missing",
			dhook.Message{Embeds: []dhook.Embed{
				{
					Description: "description",
					Fields: []dhook.EmbedField{
						{Name: "", Value: "value"},
					},
				},
			}},
			false,
		},
		// invalid embed author
		{
			"embed author name too long",
			dhook.Message{Embeds: []dhook.Embed{
				{Author: dhook.EmbedAuthor{Name: makeStr(4096)}},
				{Description: "description"},
			}},
			false,
		},
		// invalid embed footer
		{
			"embed footer too long",
			dhook.Message{Embeds: []dhook.Embed{
				{Footer: dhook.EmbedFooter{Text: makeStr(2049)}},
				{Description: "description"},
			}},
			false,
		},
		// invalid embed provider
		{
			"embed provider name too long",
			dhook.Message{Embeds: []dhook.Embed{
				{Provider: dhook.EmbedProvider{Name: makeStr(257)}},
				{Description: "description"},
			}},
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
