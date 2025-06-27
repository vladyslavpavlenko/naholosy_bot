package predicates_test

import (
	"context"
	"testing"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/config"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/handlers/predicates"
)

func TestAdmin(main *testing.T) {
	tests := []struct {
		name     string
		update   telego.Update
		expected bool
	}{
		{
			name:     "NilMessage",
			update:   telego.Update{},
			expected: false,
		},

		{
			name:     "Admin",
			update:   telego.Update{Message: &telego.Message{From: &telego.User{ID: 123}}},
			expected: true,
		},

		{
			name:     "NonAdmin",
			update:   telego.Update{Message: &telego.Message{From: &telego.User{ID: 999}}},
			expected: false,
		},
	}

	cfg := &config.Config{
		AdminIDs: []int64{123, 456},
	}

	for _, tt := range tests {
		main.Run(tt.name, func(t *testing.T) {
			predicate := predicates.Admin(cfg)
			require.Equal(t, tt.expected, predicate(context.Background(), tt.update))
		})
	}
}
