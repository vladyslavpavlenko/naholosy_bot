package handlers

import (
	"sync"
	"time"
)

// openQuiz is a question put to a user as a poll, kept until it is answered.
type openQuiz struct {
	userID int64
	// timer fires when the question has run out of time. Telegram closes the
	// poll itself but says nothing about it, so the clock has to be ours.
	timer *time.Timer
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
	// missed counts the questions that ran out of time one after another. It
	// is what tells a slow answer from someone who has walked away.
	missed map[int64]int
}

func newQuizzes() *quizzes {
	return &quizzes{
		open:    make(map[string]openQuiz),
		current: make(map[int64]string),
		missed:  make(map[int64]int),
	}
}

// timedOut counts one question that ran out and returns the run of them.
func (q *quizzes) timedOut(userID int64) int {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.missed[userID]++

	return q.missed[userID]
}

// answered clears the run of missed questions.
func (q *quizzes) answered(userID int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	delete(q.missed, userID)
}

// put records a freshly sent poll, forgetting whatever the user had open
// before so that the map does not grow with every question.
func (q *quizzes) put(pollID string, userID int64, options []string, timer *time.Timer) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if previous, ok := q.current[userID]; ok {
		q.stop(previous)
		delete(q.open, previous)
	}

	q.open[pollID] = openQuiz{userID: userID, options: options, timer: timer}
	q.current[userID] = pollID
}

// stop cancels a question's clock. The caller holds the lock.
func (q *quizzes) stop(pollID string) {
	if quiz, ok := q.open[pollID]; ok && quiz.timer != nil {
		quiz.timer.Stop()
	}
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

	q.stop(pollID)
	delete(q.open, pollID)
	delete(q.current, quiz.userID)

	return quiz, true
}

// forget drops whatever the user has open, when a run ends.
func (q *quizzes) forget(userID int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if pollID, ok := q.current[userID]; ok {
		q.stop(pollID)
		delete(q.open, pollID)
		delete(q.current, userID)
	}

	delete(q.missed, userID)
}
