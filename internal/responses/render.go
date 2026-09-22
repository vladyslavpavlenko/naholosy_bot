package responses

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/practice"
)

// QuizQuestion renders the prompt of a practice poll: the word, and the note
// that tells homographs apart. It is plain text, since the only markup a poll
// question takes is custom emoji.
//
// The note of a word that may be stressed two ways is withheld by the practice
// service, because naming it would give the answer away.
func QuizQuestion(q practice.Question) string {
	if q.Note == "" {
		return q.Word
	}

	return q.Word + "\n" + q.Note
}

// QuizProgress renders how far into the run the question is. It goes in the
// poll's description rather than the question.
func QuizProgress(q practice.Question) string {
	return fmt.Sprintf(QuizProgressPattern, q.Number, q.Total)
}

// QuizExplanation renders what Telegram shows once the answer is given: the
// correct spelling and the note that was held back during the question.
func QuizExplanation(word, note string) string {
	text := "<b>" + escape(word) + "</b>"
	if note != "" {
		text += "\n<i>" + escape(note) + "</i>"
	}

	return text
}

// Results renders the summary of a finished run, naming the words it got
// wrong so the user knows what to go back to.
func Results(summary practice.Summary) string {
	s := summary.Session
	text := fmt.Sprintf(GameResults, s.Answered, s.Size, s.Correct, s.Wrong)

	if len(summary.Missed) == 0 {
		return text
	}

	missed := make([]string, 0, len(summary.Missed))
	for _, word := range summary.Missed {
		missed = append(missed, escape(word))
	}

	return text + "\n\n" + fmt.Sprintf(MissedWords, strings.Join(missed, "\n"))
}

// WordList renders the words found for a set of letters. Asking for several
// letters at once groups the answer under each, rather than running them all
// together.
func WordList(letters []rune, accents []accent.Accent) string {
	var b strings.Builder
	fmt.Fprintf(&b, WordsHeader, escape(joinLetters(letters)))
	b.WriteString("\n\n")

	if len(accents) == 0 {
		b.WriteString(NoWordsForLetters)
		return b.String()
	}

	groups := groupByLetter(accents)
	if len(groups) == 1 {
		b.WriteString(Entries(accents))
		return b.String()
	}

	sections := make([]string, 0, len(groups))
	for _, g := range groups {
		sections = append(sections,
			fmt.Sprintf("<b>%s</b>\n%s", escape(string(unicode.ToUpper(g.letter))), Entries(g.accents)))
	}

	// Entries already ends every line, so joining leaves a blank line between
	// the groups.
	b.WriteString(strings.Join(sections, "\n"))

	return b.String()
}

// group is the words that start with one letter.
type group struct {
	letter  rune
	accents []accent.Accent
}

// groupByLetter splits words by their first letter, keeping catalog order.
func groupByLetter(accents []accent.Accent) []group {
	var groups []group
	for _, a := range accents {
		if len(groups) == 0 || groups[len(groups)-1].letter != a.Letter() {
			groups = append(groups, group{letter: a.Letter()})
		}
		last := &groups[len(groups)-1]
		last.accents = append(last.accents, a)
	}

	return groups
}

// Entries renders words one per line, each followed by its note.
func Entries(accents []accent.Accent) string {
	var b strings.Builder
	for _, a := range accents {
		b.WriteString(escape(a.Word))
		if a.Note != "" {
			b.WriteString(" <i>" + escape(a.Note) + "</i>")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// joinLetters renders the requested letters uppercased and space-separated.
func joinLetters(letters []rune) string {
	parts := make([]string, 0, len(letters))
	for _, r := range letters {
		parts = append(parts, string(unicode.ToUpper(r)))
	}

	return strings.Join(parts, " ")
}
