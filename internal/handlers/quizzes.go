package handlers

import "sync"

// openQuiz is a question put to a user as a poll, kept until it is answered.
type openQuiz struct {
	userID int64
	// options are the answers in the order they were sent, which is the only
	// way to make sense of the index a poll answer comes back with.
	options []string
}

// quizzes remembers the poll behind each open question.
//
// A poll answer carries nothing but the poll's ID and the index picked — not
// the text, not the word, not the message — so the options have to be kept on
// this side. Like the missed words, they live in the process: after a restart
// an open question can no longer be graded, and the run moves on to the next
// word instead.
type quizzes struct {
	mu      sync.Mutex
	open    map[string]openQuiz
	current map[int64]string
}

func newQuizzes() *quizzes {
	return &quizzes{open: make(map[string]openQuiz), current: make(map[int64]string)}
}

// put records a freshly sent poll, forgetting whatever the user had open
// before so that the map does not grow with every question.
func (q *quizzes) put(pollID string, userID int64, options []string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if previous, ok := q.current[userID]; ok {
		delete(q.open, previous)
	}

	q.open[pollID] = openQuiz{userID: userID, options: options}
	q.current[userID] = pollID
}

// take returns and clears the poll, so that a second answer to it finds
// nothing.
func (q *quizzes) take(pollID string) (openQuiz, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	quiz, ok := q.open[pollID]
	if !ok {
		return openQuiz{}, false
	}

	delete(q.open, pollID)
	delete(q.current, quiz.userID)

	return quiz, true
}

// forget drops whatever the user has open, when a run ends.
func (q *quizzes) forget(userID int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if pollID, ok := q.current[userID]; ok {
		delete(q.open, pollID)
		delete(q.current, userID)
	}
}
