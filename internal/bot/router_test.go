package bot_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/bot"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/config"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/handlers"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/metrics"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/responses"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
	"github.com/vladyslavpavlenko/naholosy_bot/pkg/logger"
)

func TestResolve(main *testing.T) {
	tests := []struct {
		name  string
		stage user.Stage
		text  string
		admin bool
		want  string
	}{
		{name: "Start", stage: user.StageStart, text: "/start", want: bot.RouteStart},
		{name: "StartWithBotSuffix", stage: user.StageStart, text: "/start@naholosy_bot", want: bot.RouteStart},
		{
			name:  "StartDuringGameIsIgnored",
			stage: user.StageGame, text: "/start", want: "",
		},
		{name: "FasterOpensMenu", stage: user.StageStart, text: responses.FasterButton, want: bot.RouteMenu},
		{name: "MenuFromAnywhere", stage: user.StageWords, text: responses.MenuButton, want: bot.RouteMenu},
		{
			name:  "PracticeFromMenu",
			stage: user.StageMainMenu, text: responses.PracticeButton, want: bot.RoutePracticeMenu,
		},
		{
			name:  "BackFromReadiness",
			stage: user.StagePracticeReadiness, text: responses.BackButton, want: bot.RoutePracticeMenu,
		},
		{name: "SingleLetter", stage: user.StageWords, text: "Є", want: bot.RouteLetters},
		{name: "LetterList", stage: user.StageMainMenu, text: "є я", want: bot.RouteLetters},
		{name: "WordIsNotALetterList", stage: user.StageMainMenu, text: "фольга", want: bot.RouteLookup},
		{name: "RunLength", stage: user.StagePracticeMenu, text: "24", want: bot.RouteChooseSize},
		{
			name:  "UnsupportedRunLength",
			stage: user.StagePracticeMenu, text: "13", want: bot.RouteLookup,
		},
		{
			name:  "RunLengthOnlyInPracticeMenu",
			stage: user.StageMainMenu, text: "24", want: bot.RouteLookup,
		},
		{
			name:  "Yes",
			stage: user.StagePracticeReadiness, text: responses.YesButton, want: bot.RouteStartRun,
		},
		{
			// Answers arrive as taps on the buttons under each question, so a
			// stray word typed mid-run must not be scored.
			name: "TypingDuringGameIsIgnored", stage: user.StageGame, text: "фОльга", want: "",
		},
		{
			name:  "FinishDuringGame",
			stage: user.StageGame, text: responses.FinishGameButton, want: bot.RouteFinishRun,
		},
		{
			name:  "FinishOutsideGameIsIgnored",
			stage: user.StageMainMenu, text: responses.FinishGameButton, want: bot.RouteLookup,
		},
		{
			name:  "DownloadOnlyFromMainMenu",
			stage: user.StageMainMenu, text: responses.DownloadButton, want: bot.RouteDownload,
		},
		{
			name:  "DownloadElsewhereFallsThrough",
			stage: user.StageWords, text: responses.DownloadButton, want: bot.RouteLookup,
		},
		{
			name:  "AllWordsOnlyFromMainMenu",
			stage: user.StageMainMenu, text: responses.AllWordsButton, want: bot.RouteWordsMenu,
		},
		{name: "EmptyTextDuringGameIsIgnored", stage: user.StageGame, text: "", want: ""},
		{
			name:  "MenuDuringGameIsIgnored",
			stage: user.StageGame, text: responses.MenuButton, want: "",
		},
		{name: "AdminStatus", stage: user.StageMainMenu, text: "/status", admin: true, want: bot.RouteStatus},
		{
			name:  "AdminStatusDuringGame",
			stage: user.StageGame, text: "/status", admin: true, want: bot.RouteStatus,
		},
		{
			name:  "NonAdminStatusIsJustAWord",
			stage: user.StageMainMenu, text: "/status", want: bot.RouteLookup,
		},
		{
			name:  "BroadcastTestBeatsBroadcast",
			stage: user.StageMainMenu, text: "/broadcast_test привіт", admin: true,
			want: bot.RouteBroadcastTest,
		},
		{
			name:  "Broadcast",
			stage: user.StageMainMenu, text: "/broadcast привіт", admin: true, want: bot.RouteBroadcast,
		},
		{
			name:  "BroadcastConfirm",
			stage: user.StageMainMenu, text: "/broadcast_confirm", admin: true,
			want: bot.RouteBroadcastConfirm,
		},
	}

	for _, tt := range tests {
		main.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, bot.Resolve(tt.stage, tt.text, tt.admin))
		})
	}
}

func TestParseCommand(main *testing.T) {
	tests := []struct {
		name    string
		text    string
		command string
		args    string
	}{
		{name: "Bare", text: "/status", command: "status"},
		{name: "WithArgs", text: "/broadcast привіт усім", command: "broadcast", args: "привіт усім"},
		{name: "WithBotSuffix", text: "/broadcast@naholosy_bot текст", command: "broadcast", args: "текст"},
		{name: "MixedCase", text: "/Status", command: "status"},
		{name: "TrimsArgs", text: "/broadcast    текст   ", command: "broadcast", args: "текст"},
		{
			// A broadcast spanning several lines is written with a newline
			// straight after the command.
			name: "NewlineAfterTheCommand", text: "/broadcast\n<b>Оновлення</b>\n\nДругий рядок",
			command: "broadcast", args: "<b>Оновлення</b>\n\nДругий рядок",
		},
		{name: "NewlineWithBotSuffix", text: "/broadcast@naholosy_bot\nтекст", command: "broadcast", args: "текст"},
		{name: "NotACommand", text: "фольга"},
		{name: "Empty", text: ""},
	}

	for _, tt := range tests {
		main.Run(tt.name, func(t *testing.T) {
			command, args := bot.ParseCommand(tt.text)
			require.Equal(t, tt.command, command)
			require.Equal(t, tt.args, args)
		})
	}
}

// TestEveryRouteHasAHandler guards the wiring: a route the router can return
// but the bot has no handler for would be a nil call at dispatch time.
func TestEveryRouteHasAHandler(t *testing.T) {
	b := bot.New(
		config.Config{},
		nil,
		handlers.New(nil, nil, nil, nil, nil, nil, metrics.New(), logger.New("PROD")),
		metrics.New(),
		logger.New("PROD"),
	)

	for _, route := range bot.Routes() {
		require.Truef(t, b.Handles(route), "route %q has no handler", route)
	}
}

// TestResolveOnlyReturnsKnownRoutes checks the other direction: every route the
// router can produce is one the test above covers.
func TestResolveOnlyReturnsKnownRoutes(t *testing.T) {
	known := append(bot.Routes(), "")

	texts := []string{
		"", "/start", "/status", "/broadcast", "/broadcast_test", "/broadcast_confirm",
		"/broadcast_cancel", "фольга", "є я", "12", "13",
		responses.FasterButton, responses.MenuButton, responses.BackButton,
		responses.PracticeButton, responses.YesButton, responses.AllWordsButton,
		responses.DownloadButton, responses.FinishGameButton,
	}
	stages := []user.Stage{
		user.StageStart, user.StageMainMenu, user.StageWords,
		user.StagePracticeMenu, user.StagePracticeReadiness, user.StageGame,
	}

	for _, stage := range stages {
		for _, text := range texts {
			for _, admin := range []bool{false, true} {
				require.Containsf(t, known, bot.Resolve(stage, text, admin),
					"stage %q, text %q", stage, text)
			}
		}
	}
}
