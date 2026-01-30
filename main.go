package emojiparser

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

//go:embed assets/*.json
var assetsFS embed.FS

// Assets holds the parsed emoji lookup tables.
type Assets struct {
	UnicodeEmojis    map[string]string
	UnicodeEmojisSVG map[string]string
}

// EmojiPosition represents the start and end indices of an emoji in the source string.
type EmojiPosition struct {
	From int
	To   int
}

// EmojiType represents the type of emoji.
type EmojiType string

const (
	EmojiTypeUnicode EmojiType = "unicode"
	EmojiTypeText    EmojiType = "text"
	EmojiTypeCustom  EmojiType = "custom"
)

// ParsedEmoji represents a parsed emoji entry.
type ParsedEmoji struct {
	ID       *string
	Name     string
	Type     EmojiType
	Unicode  string
	Position EmojiPosition
	Link     *string
	Animated bool
}

// DiscordEmojiParser parses unicode, text, and custom emojis from a string.
type DiscordEmojiParser struct {
	assets        *Assets
	nameToUnicode map[string]string
	unicodeToName map[string]string
	unicodeKeys   []string
	customRegex   *regexp.Regexp
	textRegex     *regexp.Regexp
}

var defaultParser *DiscordEmojiParser

func init() {
	parser, err := NewDiscordEmojiParser()
	if err != nil {
		panic(err)
	}
	defaultParser = parser
}

// Parse parses all emoji types using the default parser.
func Parse(content string) []ParsedEmoji {
	return defaultParser.Parse(content)
}

// ParseUnicode parses unicode emojis using the default parser.
func ParseUnicode(content string, skipRanges []ParsedEmoji) []ParsedEmoji {
	return defaultParser.ParseUnicode(content, skipRanges)
}

// ParseTextRepresentation parses text emoji representations using the default parser.
func ParseTextRepresentation(content string, skipRanges []ParsedEmoji) []ParsedEmoji {
	return defaultParser.ParseTextRepresentation(content, skipRanges)
}

// ParseDiscordCustom parses custom emojis using the default parser.
func ParseDiscordCustom(content string) []ParsedEmoji {
	return defaultParser.ParseDiscordCustom(content)
}

// parseAssets loads and parses all JSON files under assets/.
// It returns the parsed emoji maps or an error.
func parseAssets() (*Assets, error) {
	unicodeEmojis, err := parseJSONMap("assets/UnicodeEmojis.json")
	if err != nil {
		return nil, fmt.Errorf("parse assets/UnicodeEmojis.json: %w", err)
	}

	unicodeEmojisSVG, err := parseJSONMap("assets/UnicodeEmojisSVG.json")
	if err != nil {
		return nil, fmt.Errorf("parse assets/UnicodeEmojisSVG.json: %w", err)
	}

	return &Assets{
		UnicodeEmojis:    unicodeEmojis,
		UnicodeEmojisSVG: unicodeEmojisSVG,
	}, nil
}

// NewDiscordEmojiParser creates a new parser instance with embedded assets.
func NewDiscordEmojiParser() (*DiscordEmojiParser, error) {
	assets, err := parseAssets()
	if err != nil {
		return nil, err
	}

	nameToUnicode := make(map[string]string)
	unicodeToName := make(map[string]string)
	for key, value := range assets.UnicodeEmojis {
		if containsNonASCII(key) {
			unicodeToName[key] = value
		}
		if containsNonASCII(value) {
			nameToUnicode[key] = value
		}
	}

	unicodeKeys := make([]string, 0, len(unicodeToName))
	for key := range unicodeToName {
		unicodeKeys = append(unicodeKeys, key)
	}
	sort.Slice(unicodeKeys, func(i, j int) bool {
		return len(unicodeKeys[i]) > len(unicodeKeys[j])
	})

	return &DiscordEmojiParser{
		assets:        assets,
		nameToUnicode: nameToUnicode,
		unicodeToName: unicodeToName,
		unicodeKeys:   unicodeKeys,
		customRegex:   regexp.MustCompile(`<(a?):(\w+):(\d{16,})>`),
		textRegex:     regexp.MustCompile(`:([A-Za-z0-9_]+):`),
	}, nil
}

// Parse parses all emoji types from the provided content.
func (p *DiscordEmojiParser) Parse(content string) []ParsedEmoji {
	customEmojis := p.ParseDiscordCustom(content)
	unicodeEmojis := p.ParseUnicode(content, customEmojis)
	textEmojis := p.ParseTextRepresentation(content, customEmojis)

	all := append(append(unicodeEmojis, textEmojis...), customEmojis...)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Position.From < all[j].Position.From
	})
	return all
}

// ParseUnicode parses unicode emojis from the content.
func (p *DiscordEmojiParser) ParseUnicode(content string, skipRanges []ParsedEmoji) []ParsedEmoji {
	results := make([]ParsedEmoji, 0)
	for i := 0; i < len(content); {
		if p.isInsideRange(i, skipRanges) {
			_, size := utf8.DecodeRuneInString(content[i:])
			i += size
			continue
		}

		match := ""
		for _, key := range p.unicodeKeys {
			if strings.HasPrefix(content[i:], key) {
				match = key
				break
			}
		}

		if match == "" {
			_, size := utf8.DecodeRuneInString(content[i:])
			i += size
			continue
		}

		from := i
		to := i + len(match)
		if p.isInsideRange(from, skipRanges) {
			i = to
			continue
		}

		name := p.unicodeToName[match]
		codePoint := toCodePoint(match, "-")
		var link *string
		if hash, ok := p.assets.UnicodeEmojisSVG[codePoint]; ok {
			url := "https://discord.com/assets/" + hash
			link = &url
		}

		results = append(results, ParsedEmoji{
			ID:       nil,
			Name:     name,
			Type:     EmojiTypeUnicode,
			Unicode:  match,
			Position: EmojiPosition{From: from, To: to},
			Link:     link,
			Animated: false,
		})
		i = to
	}

	return results
}

// ParseTextRepresentation parses text emoji representations like :smile: from content.
func (p *DiscordEmojiParser) ParseTextRepresentation(content string, skipRanges []ParsedEmoji) []ParsedEmoji {
	results := make([]ParsedEmoji, 0)
	matches := p.textRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		from := match[0]
		to := match[1]
		name := content[match[2]:match[3]]

		if p.isInsideRange(from, skipRanges) {
			continue
		}
		unicode, ok := p.nameToUnicode[name]
		if !ok {
			continue
		}

		codePoint := toCodePoint(unicode, "-")
		var link *string
		if hash, ok := p.assets.UnicodeEmojisSVG[codePoint]; ok {
			url := "https://discord.com/assets/" + hash + ".svg"
			link = &url
		}

		results = append(results, ParsedEmoji{
			ID:       nil,
			Name:     name,
			Type:     EmojiTypeText,
			Unicode:  unicode,
			Position: EmojiPosition{From: from, To: to},
			Link:     link,
			Animated: false,
		})
	}

	return results
}

// ParseDiscordCustom parses custom Discord emojis like <:name:id> or <a:name:id>.
func (p *DiscordEmojiParser) ParseDiscordCustom(content string) []ParsedEmoji {
	results := make([]ParsedEmoji, 0)
	matches := p.customRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}
		from := match[0]
		to := match[1]
		animatedFlag := content[match[2]:match[3]]
		name := content[match[4]:match[5]]
		id := content[match[6]:match[7]]

		animated := animatedFlag == "a"
		ext := "png"
		if animated {
			ext = "gif"
		}
		url := "https://cdn.discordapp.com/emojis/" + id + "." + ext

		idCopy := id
		results = append(results, ParsedEmoji{
			ID:       &idCopy,
			Name:     name,
			Type:     EmojiTypeCustom,
			Unicode:  content[from:to],
			Position: EmojiPosition{From: from, To: to},
			Link:     &url,
			Animated: animated,
		})
	}

	return results
}

func toCodePoint(str, sep string) string {
	points := make([]string, 0)
	for _, r := range str {
		points = append(points, fmt.Sprintf("%x", r))
	}
	return strings.Join(points, sep)
}

func (p *DiscordEmojiParser) isInsideRange(index int, ranges []ParsedEmoji) bool {
	for _, item := range ranges {
		if index >= item.Position.From && index < item.Position.To {
			return true
		}
	}
	return false
}

func containsNonASCII(value string) bool {
	for _, r := range value {
		if r > 127 {
			return true
		}
	}
	return false
}

func parseJSONMap(path string) (map[string]string, error) {
	file, err := assetsFS.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var result map[string]string
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, err
	}

	return result, nil
}
