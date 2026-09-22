// Package practice runs the training loop: a user commits to a number of
// words, the bot asks them one at a time, and the run is graded at the end.
package practice

import (
	"fmt"
	"slices"
)

// Sizes are the run lengths a user may choose from.
var Sizes = []int{12, 24, 36}

// ValidSize reports whether n is an offered run length.
func ValidSize(n int) bool { return slices.Contains(Sizes, n) }

// Grade is how well a completed run went.
type Grade int

const (
	// GradePoor is at most a third answered correctly.
	GradePoor Grade = iota
	// GradeFair is more than a third, short of a near-perfect run.
	GradeFair
	// GradeExcellent is within [excellentMargin] of a clean sweep.
	GradeExcellent
)

const (
	excellentMargin = 3
	poorFraction    = 3
)

// Session is one practice run. It lives on the user's row, so it has no
// identity of its own.
type Session struct {
	UserID   int64
	Size     int
	Answered int
	Correct  int
	Wrong    int
	// CurrentWord is the accented word being asked right now, empty between
	// questions.
	CurrentWord string
	// AskedWords is every word this run has already used.
	AskedWords []string
}

// NewSession starts a run of the given length.
func NewSession(userID int64, size int) (*Session, error) {
	if !ValidSize(size) {
		return nil, fmt.Errorf("practice: unsupported run length %d", size)
	}

	return &Session{UserID: userID, Size: size, AskedWords: make([]string, 0, size)}, nil
}

// Ask records that the user is now being asked about word.
func (s *Session) Ask(word string) {
	s.CurrentWord = word
	if !slices.Contains(s.AskedWords, word) {
		s.AskedWords = append(s.AskedWords, word)
	}
}

// Asked returns the words this run has used, for excluding them from the next
// draw.
func (s *Session) Asked() map[string]struct{} {
	asked := make(map[string]struct{}, len(s.AskedWords))
	for _, w := range s.AskedWords {
		asked[w] = struct{}{}
	}

	return asked
}

// Answer records a reply and reports whether it was right. The current
// question is cleared either way, so a duplicate reply is ignored rather than
// counted twice.
//
// A word answered wrongly is dropped from the asked set, so the run may put it
// again later. Getting something wrong and never seeing it again is the worst
// thing a drill can do.
func (s *Session) Answer(given string) (correct, ok bool) {
	if s.CurrentWord == "" {
		return false, false
	}

	word := s.CurrentWord
	correct = given == word
	s.CurrentWord = ""
	s.Answered++

	if correct {
		s.Correct++
	} else {
		s.Wrong++
		s.AskedWords = slices.DeleteFunc(s.AskedWords, func(w string) bool { return w == word })
	}

	return correct, true
}

// Complete reports whether the run has had all the answers it asked for.
func (s *Session) Complete() bool { return s.Answered >= s.Size }

// QuestionNumber is the 1-based position of the question being asked now.
func (s *Session) QuestionNumber() int { return s.Answered + 1 }

// Grade rates the run.
func (s *Session) Grade() Grade {
	switch {
	case s.Correct <= s.Size/poorFraction:
		return GradePoor
	case s.Correct < s.Size-excellentMargin:
		return GradeFair
	default:
		return GradeExcellent
	}
}
