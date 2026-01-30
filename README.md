# Discord Emoji Parser for Go

Parse Discord emoji content with unicode, text, and custom emoji support.

## Install

```
go get github.com/x1xo/emoji-parser
```

## Usage

The package initializes a default parser on import (it panics if embedded assets fail to load).

```go
package main

import (
	"fmt"

	emojiparser "github.com/x1xo/emoji-parser"
)

func main() {
	content := "Hello 😄 :smile: <a:wave:1234567890123456>"
	parsed := emojiparser.Parse(content)

	for _, emoji := range parsed {
		fmt.Printf("%s %s %v %v\n", emoji.Type, emoji.Name, emoji.Position, emoji.Link)
	}
}
```

### Parse unicode emojis

```go
results := emojiparser.ParseUnicode("ok 😄!", nil)
```

### Parse text emojis

```go
results := emojiparser.ParseTextRepresentation("hi :smile:", nil)
```

### Parse custom emojis

Custom emoji IDs must be at least 16 digits.

```go
results := emojiparser.ParseDiscordCustom("<a:wave:1234567890123456>")
```

Note: Another validation is required to check if that emoji exists within Discord.

## ParsedEmoji

`ParsedEmoji` includes:

- `ID` (only for custom emojis)
- `Name`
- `Type` (`unicode`, `text`, `custom`)
- `Unicode` (original match)
- `Position` (`From`, `To` byte indexes)
- `Link` (Discord asset URL when available)
- `Animated` (custom emoji only)

## Notes

- Asset files are embedded from `assets/*.json`.
- The default parser is created at package init and will panic if assets cannot be loaded.