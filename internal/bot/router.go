package bot

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/words"
)

// Route names. They identify the handler to run and label the metrics.
const (
	RouteStart            = "start"
	RouteMenu             = "menu"
	RouteWordsMenu        = "words.menu"
	RouteLetters          = "words.letters"
	RouteLookup           = "words.lookup"
	RouteDownload         = "download"
	RoutePracticeMenu     = "practice.menu"
	RouteChooseSize       = "practice.size"
	RouteStartRun         = "practice.start"
	RouteAnswer           = "practice.answer"
	RouteTimeout          = "practice.timeout"
	RouteFinishRun        = "practice.finish"
	RouteStatus           = "admin.status"
	RouteBroadcast        = "admin.broadcast"
	RouteBroadcastTest    = "admin.broadcast_test"
	RouteBroadcastConfirm = "admin.broadcast_confirm"
	RouteBroadcastCancel  = "admin.broadcast_cancel"
)

// adminRoutes maps an admin command to its route.
var adminRoutes = map[string]string{
	"status":            RouteStatus,
	"broadcast":         RouteBroadcast,
	"broadcast_test":    RouteBroadcastTest,
	"broadcast_confirm": RouteBroadcastConfirm,
	"broadcast_cancel":  RouteBroadcastCancel,
}

// Resolve picks the handler for a message. It mirrors the original bot's
// dispatch order, where the same button means different things depending on
// the stage. An empty route means the message is ignored.
func Resolve(stage user.Stage, text string, admin bool) string {
	command, _ := ParseCommand(text)

	if admin {
		if route, ok := adminRoutes[command]; ok {
			return route
		}
	}

	// A run is driven by the buttons under each question, which arrive as
	// callbacks rather than messages. Nothing else typed during one means
	// anything, and treating it as an answer would only score a stray word.
	if stage == user.StageGame {
		if text == responses.FinishGameButton {
			return RouteFinishRun
		}

		return ""
	}

	switch {
	case command == "start":
		return RouteStart
	case text == responses.FasterButton || text == responses.MenuButton:
		return RouteMenu
	case text == responses.PracticeButton || text == responses.BackButton:
		return RoutePracticeMenu
	case words.IsLetterList(text):
		return RouteLetters
	case stage == user.StagePracticeMenu && isRunLength(text):
		return RouteChooseSize
	case stage == user.StagePracticeReadiness && text == responses.YesButton:
		return RouteStartRun
	case stage == user.StageMainMenu && text == responses.DownloadButton:
		return RouteDownload
	case stage == user.StageMainMenu && text == responses.AllWordsButton:
		return RouteWordsMenu
	default:
		return RouteLookup
	}
}

// ParseCommand splits "/broadcast@bot some text" into "broadcast" and
// "some text". A message that is not a command yields an empty name.
//
// The split is at the first whitespace rather than the first space, because a
// broadcast spanning several lines is written with a newline straight after
// the command and would otherwise not be recognized as one at all.
func ParseCommand(text string) (command, args string) {
	if !strings.HasPrefix(text, "/") {
		return "", ""
	}

	rest := strings.TrimPrefix(text, "/")

	end := strings.IndexFunc(rest, unicode.IsSpace)
	if end < 0 {
		end = len(rest)
	}

	command, _, _ = strings.Cut(rest[:end], "@")

	return strings.ToLower(command), strings.TrimSpace(rest[end:])
}

func isRunLength(text string) bool {
	size, err := strconv.Atoi(text)

	return err == nil && practice.ValidSize(size)
}
