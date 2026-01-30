package emojiparser_test

import (
	"strings"
	"testing"

	emojiparser "github.com/x1xo/emoji-parser"
)

func TestParseUnicode(t *testing.T) {
	content := "ok 😄!"
	results := emojiparser.ParseUnicode(content, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 unicode emoji, got %d", len(results))
	}
	result := results[0]
	if result.Name != "smile" {
		t.Fatalf("expected name smile, got %s", result.Name)
	}
	if result.Type != emojiparser.EmojiTypeUnicode {
		t.Fatalf("expected type unicode, got %s", result.Type)
	}
	from := strings.Index(content, "😄")
	if result.Position.From != from {
		t.Fatalf("expected from %d, got %d", from, result.Position.From)
	}
	if result.Position.To != from+len("😄") {
		t.Fatalf("expected to %d, got %d", from+len("😄"), result.Position.To)
	}
	if result.Link == nil || !strings.HasPrefix(*result.Link, "https://discord.com/assets/") {
		t.Fatalf("expected unicode link to discord assets, got %v", result.Link)
	}
}

func TestParseTextRepresentation(t *testing.T) {
	content := "hi :smile:"
	results := emojiparser.ParseTextRepresentation(content, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 text emoji, got %d", len(results))
	}
	result := results[0]
	if result.Name != "smile" {
		t.Fatalf("expected name smile, got %s", result.Name)
	}
	if result.Type != emojiparser.EmojiTypeText {
		t.Fatalf("expected type text, got %s", result.Type)
	}
	from := strings.Index(content, ":smile:")
	if result.Position.From != from {
		t.Fatalf("expected from %d, got %d", from, result.Position.From)
	}
	if result.Position.To != from+len(":smile:") {
		t.Fatalf("expected to %d, got %d", from+len(":smile:"), result.Position.To)
	}
	if result.Unicode != "😄" {
		t.Fatalf("expected unicode 😄, got %s", result.Unicode)
	}
	if result.Link == nil || !strings.HasSuffix(*result.Link, ".svg") {
		t.Fatalf("expected svg link, got %v", result.Link)
	}
}

func TestParseDiscordCustom(t *testing.T) {
	content := "hello <a:wave:1234567890123456> and <:smile:6789012345678901>"
	results := emojiparser.ParseDiscordCustom(content)
	if len(results) != 2 {
		t.Fatalf("expected 2 custom emojis, got %d", len(results))
	}
	if results[0].Name != "wave" || !results[0].Animated {
		t.Fatalf("expected first custom emoji to be animated wave")
	}
	if results[0].ID == nil || *results[0].ID != "1234567890123456" {
		t.Fatalf("expected first custom emoji id 1234567890123456")
	}
	if results[0].Link == nil || !strings.HasSuffix(*results[0].Link, ".gif") {
		t.Fatalf("expected gif link for animated emoji")
	}
	if results[1].Name != "smile" || results[1].Animated {
		t.Fatalf("expected second custom emoji to be static smile")
	}
	if results[1].ID == nil || *results[1].ID != "6789012345678901" {
		t.Fatalf("expected second custom emoji id 6789012345678901")
	}
	if results[1].Link == nil || !strings.HasSuffix(*results[1].Link, ".png") {
		t.Fatalf("expected png link for static emoji")
	}
}

func TestParseAllSorted(t *testing.T) {
	content := "A :smile: B 😄 C <a:wave:1234567890123456>"
	results := emojiparser.Parse(content)
	if len(results) != 3 {
		t.Fatalf("expected 3 emojis, got %d", len(results))
	}
	for i := 1; i < len(results); i++ {
		if results[i-1].Position.From > results[i].Position.From {
			t.Fatalf("results not sorted by position")
		}
	}
}
