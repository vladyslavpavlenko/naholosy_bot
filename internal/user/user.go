// Package user holds the people talking to the bot and where they are in the
// conversation.
package user

// User is a Telegram user known to the bot.
type User struct {
	ID int64
	// Stage is the state the user's next message is interpreted in.
	Stage Stage
	// TutorialSeen records that the practice rules were explained once.
	TutorialSeen bool
}

// MoveTo puts the user into a new stage.
func (u *User) MoveTo(stage Stage) { u.Stage = stage }

// SeeTutorial marks the practice rules as explained.
func (u *User) SeeTutorial() { u.TutorialSeen = true }
