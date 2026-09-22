package user_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/user"
)

func TestParseStage(main *testing.T) {
	main.Run("KnownStages", func(t *testing.T) {
		for _, stage := range []user.Stage{
			user.StageStart, user.StageMainMenu, user.StageWords,
			user.StagePracticeMenu, user.StagePracticeReadiness, user.StageGame,
		} {
			require.Equal(t, stage, user.ParseStage(stage.String()))
		}
	})

	main.Run("UnknownFallsBackToStart", func(t *testing.T) {
		// The old bot wrote stage names without validation, so these really
		// do turn up in the database.
		for _, raw := range []string{"", "main-menu", "GAME", "{[]}"} {
			require.Equal(t, user.StageStart, user.ParseStage(raw))
		}
	})
}

func TestMoveTo(t *testing.T) {
	u := &user.User{Stage: user.StageStart}
	u.MoveTo(user.StageGame)
	require.Equal(t, user.StageGame, u.Stage)
}

func TestSeeTutorial(t *testing.T) {
	u := &user.User{}
	require.False(t, u.TutorialSeen)

	u.SeeTutorial()
	require.True(t, u.TutorialSeen)
}
