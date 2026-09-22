package user

// Stage is where a user currently is in the conversation. The same message
// text means different things depending on it.
type Stage string

const (
	// StageStart is a user who has been greeted but has not opened the menu.
	StageStart Stage = "start"
	// StageMainMenu is the main menu, where free-form words are looked up.
	StageMainMenu Stage = "main_menu"
	// StageWords is the word browser, driven by the alphabet keyboard.
	StageWords Stage = "words"
	// StagePracticeMenu is where the length of a practice run is chosen.
	StagePracticeMenu Stage = "practice_menu"
	// StagePracticeReadiness is the confirmation shown before a run starts.
	StagePracticeReadiness Stage = "practice_readiness"
	// StageGame is an in-progress practice run.
	StageGame Stage = "game"
)

// stages is the set of values that may be persisted.
var stages = map[Stage]struct{}{
	StageStart:             {},
	StageMainMenu:          {},
	StageWords:             {},
	StagePracticeMenu:      {},
	StagePracticeReadiness: {},
	StageGame:              {},
}

// ParseStage converts a stored value into a Stage, falling back to
// [StageStart]. The old bot wrote stage names without validation, so unknown
// values do occur and must not wedge a user.
func ParseStage(s string) Stage {
	stage := Stage(s)
	if _, ok := stages[stage]; ok {
		return stage
	}

	return StageStart
}

// String implements [fmt.Stringer].
func (s Stage) String() string { return string(s) }
