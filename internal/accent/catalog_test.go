package accent_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vladyslavpavlenko/naholosy_bot/internal/accent"
)

// sample mirrors the shape of the real list: a double-stressed word, a pair of
// homographs, an apostrophe and a hyphen.
func sample() []accent.Accent {
	return []accent.Accent{
		{ID: 1, Word: "алфАвІт", Note: "(подвійний наголос)"},
		{ID: 2, Word: "вИгода", Note: "(користь)"},
		{ID: 3, Word: "вигОда", Note: "(зручність)"},
		{ID: 4, Word: "де-Юре"},
		{ID: 5, Word: "тім'янИй"},
		{ID: 6, Word: "фОльга"},
	}
}

func newCatalog(t *testing.T) *accent.Catalog {
	t.Helper()

	catalog, err := accent.NewCatalog(sample())
	require.NoError(t, err)

	return catalog
}

func TestNewCatalog(main *testing.T) {
	main.Run("RejectsEmpty", func(t *testing.T) {
		catalog, err := accent.NewCatalog(nil)
		require.Error(t, err)
		require.Nil(t, catalog)
	})

	main.Run("RejectsUnusableWord", func(t *testing.T) {
		_, err := accent.NewCatalog([]accent.Accent{{Word: "hello"}})
		require.ErrorContains(t, err, "hello")
	})
}

func TestLookup(main *testing.T) {
	catalog := newCatalog(main)

	main.Run("Found", func(t *testing.T) {
		require.Len(t, catalog.Lookup("фольга"), 1)
	})

	main.Run("CaseInsensitive", func(t *testing.T) {
		require.Len(t, catalog.Lookup("ФОЛЬГА"), 1)
	})

	main.Run("Homographs", func(t *testing.T) {
		found := catalog.Lookup("вигода")
		require.Len(t, found, 2)
		require.Equal(t, "вИгода", found[0].Word)
		require.Equal(t, "вигОда", found[1].Word)
	})

	main.Run("ApostropheVariantsMatch", func(t *testing.T) {
		require.Len(t, catalog.Lookup("тім’яний"), 1)
		require.Len(t, catalog.Lookup("тім'яний"), 1)
	})

	main.Run("Unknown", func(t *testing.T) {
		require.Empty(t, catalog.Lookup("невідомо"))
	})
}

func TestLetters(t *testing.T) {
	require.Equal(t, []rune{'а', 'в', 'д', 'т', 'ф'}, newCatalog(t).Letters())
}

func TestByLetters(main *testing.T) {
	catalog := newCatalog(main)

	main.Run("KeepsCatalogOrder", func(t *testing.T) {
		found := catalog.ByLetters([]rune{'ф', 'а'})
		require.Equal(t, []string{"алфАвІт", "фОльга"}, wordsOf(found))
	})

	main.Run("UnknownLetter", func(t *testing.T) {
		require.Empty(t, catalog.ByLetters([]rune{'ю'}))
	})

	main.Run("NoLetters", func(t *testing.T) {
		require.Empty(t, catalog.ByLetters(nil))
	})
}

func TestRandomExcept(main *testing.T) {
	catalog := newCatalog(main)

	main.Run("NeverDrawsExcluded", func(t *testing.T) {
		exclude := map[string]struct{}{}
		for _, a := range sample()[:len(sample())-1] {
			exclude[a.Word] = struct{}{}
		}

		for range 50 {
			drawn, ok := catalog.RandomExcept(exclude)
			require.True(t, ok)
			require.Equal(t, "фОльга", drawn.Word)
		}
	})

	main.Run("ReportsExhaustion", func(t *testing.T) {
		exclude := map[string]struct{}{}
		for _, a := range sample() {
			exclude[a.Word] = struct{}{}
		}

		_, ok := catalog.RandomExcept(exclude)
		require.False(t, ok)
	})
}

func TestRandomAmong(main *testing.T) {
	catalog := newCatalog(main)

	main.Run("DrawsFromCandidates", func(t *testing.T) {
		for range 50 {
			drawn, ok := catalog.RandomAmong([]string{"фОльга", "вИгода"}, nil)
			require.True(t, ok)
			require.Contains(t, []string{"фОльга", "вИгода"}, drawn.Word)
		}
	})

	main.Run("IgnoresUnknownCandidates", func(t *testing.T) {
		drawn, ok := catalog.RandomAmong([]string{"немає", "фОльга"}, nil)
		require.True(t, ok)
		require.Equal(t, "фОльга", drawn.Word)
	})

	main.Run("EmptyWhenAllExcluded", func(t *testing.T) {
		_, ok := catalog.RandomAmong([]string{"фОльга"}, map[string]struct{}{"фОльга": {}})
		require.False(t, ok)
	})

	main.Run("EmptyCandidates", func(t *testing.T) {
		_, ok := catalog.RandomAmong(nil, nil)
		require.False(t, ok)
	})
}

func TestAccent(main *testing.T) {
	main.Run("Plain", func(t *testing.T) {
		require.Equal(t, "тім'яний", accent.Accent{Word: "тім’янИй"}.Plain())
	})

	main.Run("Letter", func(t *testing.T) {
		require.Equal(t, 'д', accent.Accent{Word: "де-Юре"}.Letter())
	})

	main.Run("DoubleStressed", func(t *testing.T) {
		require.True(t, accent.Accent{Word: "алфАвІт"}.DoubleStressed())
		require.False(t, accent.Accent{Word: "фОльга"}.DoubleStressed())
	})
}

func wordsOf(accents []accent.Accent) []string {
	out := make([]string, 0, len(accents))
	for _, a := range accents {
		out = append(out, a.Word)
	}

	return out
}
