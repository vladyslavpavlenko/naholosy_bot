package responses

import "strings"

// escaper escapes exactly the three characters Telegram's HTML parser treats
// as markup. The standard library's html.EscapeString also escapes quotes and
// apostrophes, which would turn "тім'янИй" into "тім&#39;янИй" in the source
// of every message for no benefit.
var escaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// escape makes text safe to interpolate into an HTML message.
func escape(text string) string {
	return escaper.Replace(text)
}
