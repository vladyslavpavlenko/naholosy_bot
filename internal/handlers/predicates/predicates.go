package predicates

import (
	"context"
	"slices"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/config"
)

// Admin is true if the message is sent by an admin.
func Admin(app *config.Config) th.Predicate {
	return func(_ context.Context, u telego.Update) bool {
		if u.Message == nil {
			return false
		}
		return slices.Contains(app.AdminIDs, u.Message.From.ID)
	}
}
