package responses

import (
	"math/rand/v2"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
)

// Emoji sent when a run is over, by how well it went.
var gradeEmoji = [][]string{
	practice.GradePoor:      {"😔", "🥴", "😕"},
	practice.GradeFair:      {"🤔", "🤕", "🤧"},
	practice.GradeExcellent: {"🥳", "😍", "🤩", "😎"},
}

// Reaction picks a response to a finished run.
func Reaction(grade practice.Grade) string {
	i := int(grade)
	if i < 0 || i >= len(gradeEmoji) {
		i = int(practice.GradePoor)
	}

	return gradeEmoji[i][rand.IntN(len(gradeEmoji[i]))] //nolint:gosec // not a security decision
}
