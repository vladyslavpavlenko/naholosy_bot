package handlers

import "time"

// SetQuestionClock shortens how long a question lives, so that tests covering
// the timeout do not have to sit through the real seven seconds.
func (h *Handlers) SetQuestionClock(timeout, grace time.Duration) {
	h.clock = questionClock{timeout: timeout, grace: grace}
}
