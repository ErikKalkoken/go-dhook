package dhook

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Error representing an invalid message, e.g. a message with fields that are too long.
var ErrInvalidMessage = errors.New("invalid message")

// Discord message limit
const (
	authorNameLength    = 256
	contentLength       = 2000
	descriptionLength   = 4096
	embedCombinedLength = 6000
	embedsQuantity      = 10
	fieldNameLength     = 256
	fieldsQuantity      = 25
	fieldValueLength    = 1024
	footerTextLength    = 2048
	titleLength         = 256
	usernameLength      = 80
)

// Message represents a message that can be send to a Discord webhook.
type Message struct {
	AllowedMentions bool    `json:"allowed_mentions,omitempty"`
	AvatarURL       string  `json:"avatar_url,omitempty"`
	Content         string  `json:"content,omitempty"`
	Embeds          []Embed `json:"embeds,omitempty"`
	Username        string  `json:"username,omitempty"`
}

// Validate checks the message against known Discord limits and requirements.
// Messages not passing the validation will usually lead to 400 Bad Request responses from Discord.
// Returns an [ErrInvalidMessage] error in case a limit is violated.
func (m Message) Validate() error {
	if len(m.Content) == 0 && len(m.Embeds) == 0 {
		return fmt.Errorf("need to contain content or embeds: %w", ErrInvalidMessage)
	}
	if length(m.Content) > contentLength {
		return fmt.Errorf("content too long: %w", ErrInvalidMessage)
	}
	if length(m.Username) > usernameLength {
		return fmt.Errorf("username too long: %w", ErrInvalidMessage)
	}
	if len(m.Embeds) > embedsQuantity {
		return fmt.Errorf("too many embeds: %w", ErrInvalidMessage)
	}
	var totalSize int
	for _, em := range m.Embeds {
		if err := em.validate(); err != nil {
			return err
		}
		totalSize += em.size()
	}
	if totalSize > embedCombinedLength {
		return fmt.Errorf("too many characters in combined embeds: %w", ErrInvalidMessage)
	}
	return nil
}

// Embed represents a Discord Embed.
type Embed struct {
	Author      EmbedAuthor    `json:"author,omitempty"`
	Color       int            `json:"color,omitempty"`
	Description string         `json:"description,omitempty"`
	Fields      []EmbedField   `json:"fields,omitempty"`
	Footer      EmbedFooter    `json:"footer,omitempty"`
	Image       EmbedImage     `json:"image,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Title       string         `json:"title,omitempty"`
	Thumbnail   EmbedThumbnail `json:"thumbnail,omitempty"`
	URL         string         `json:"url,omitempty"`
}

func (em Embed) size() int {
	x := length(em.Title) + length(em.Description) + length(em.Author.Name) + length(em.Footer.Text)
	for _, f := range em.Fields {
		x += f.size()
	}
	return x
}

func (em Embed) validate() error {
	em.Author.validate()
	if length(em.Description) > descriptionLength {
		return fmt.Errorf("embed description too long: %w", ErrInvalidMessage)
	}
	em.Footer.validate()
	if len(em.Fields) > fieldsQuantity {
		return fmt.Errorf("embed has too many fields: %w", ErrInvalidMessage)
	}
	for _, f := range em.Fields {
		if err := f.validate(); err != nil {
			return err
		}
	}
	if length(em.Title) > titleLength {
		return fmt.Errorf("embed title too long: %w", ErrInvalidMessage)
	}
	if em.Timestamp != "" {
		_, err := time.Parse(time.RFC3339, em.Timestamp)
		if err != nil {
			return fmt.Errorf("embed timestamp does not conform to RFC3339: %w", ErrInvalidMessage)
		}
	}
	if err := em.Author.validate(); err != nil {
		return err
	}
	if err := em.Footer.validate(); err != nil {
		return err
	}
	if err := em.Image.validate(); err != nil {
		return err
	}
	if err := em.Thumbnail.validate(); err != nil {
		return err
	}
	return nil
}

// EmbedAuthor represents the author in an [Embed].
type EmbedAuthor struct {
	Name    string `json:"name,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
	URL     string `json:"url,omitempty"`
}

func (ea EmbedAuthor) validate() error {
	if length(ea.Name) > authorNameLength {
		return fmt.Errorf("embed author name too long: %w", ErrInvalidMessage)
	}
	ok, err := isValidPublicURL(ea.IconURL)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("embed author icon URL not valid: %w", ErrInvalidMessage)
	}
	ok, err = isValidPublicURL(ea.URL)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("embed author URL not valid: %w", ErrInvalidMessage)
	}
	return nil
}

// EmbedField represents a field in an [Embed].
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

func (ef EmbedField) size() int {
	return length(ef.Name) + length(ef.Value)
}

func (ef EmbedField) validate() error {
	if ef.Name == "" {
		return fmt.Errorf("embed field name not defined: %w", ErrInvalidMessage)
	}
	if length(ef.Name) > fieldNameLength {
		return fmt.Errorf("embed field name too long: %w", ErrInvalidMessage)
	}
	if length(ef.Value) > fieldNameLength {
		return fmt.Errorf("embed field value too long: %w", ErrInvalidMessage)
	}
	return nil
}

// EmbedAuthor represents the footer of an [Embed].
type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

func (ef EmbedFooter) validate() error {
	if length(ef.Text) > footerTextLength {
		return fmt.Errorf("embed footer text too long: %w", ErrInvalidMessage)
	}
	ok, err := isValidPublicURL(ef.IconURL)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("footer icon URL not valid: %w", ErrInvalidMessage)
	}
	return nil
}

// EmbedAuthor represents the image in an [Embed].
type EmbedImage struct {
	URL string `json:"url,omitempty"`
}

func (ei EmbedImage) validate() error {
	ok, err := isValidPublicURL(ei.URL)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("embed image URL not valid: %w", ErrInvalidMessage)
	}
	return nil
}

// EmbedAuthor represents the thumbnail image in an [Embed].
type EmbedThumbnail struct {
	URL string `json:"url,omitempty"`
}

func (et EmbedThumbnail) validate() error {
	ok, err := isValidPublicURL(et.URL)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("embed thumbnail URL not valid: %w", ErrInvalidMessage)
	}
	return nil
}

// length returns the number of runes in a string.
func length(s string) int {
	return len([]rune(s))
}

// isValidPublicURL reports whether a raw URL is both a public URL and valid.
func isValidPublicURL(rawURL string) (bool, error) {
	if rawURL == "" {
		return true, nil
	}
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false, fmt.Errorf("parsing public URL: %s: %w", rawURL, err)
	}
	if u.Scheme == "https" || u.Scheme == "http" {
		return true, nil
	}
	return false, nil
}
