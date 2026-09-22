package words

import "math/rand/v2"

// AnswerVariants returns the options to offer for a correctly accented word:
// every single-accent variant in vowel order, followed by doubles of
// double-accent ones. The correct answer is always among them.
//
// A handful of words carry a legitimate double stress ("алфАвІт") that no
// single-vowel variant can reach. Offering a double-accent option only when it
// happens to be the answer would give the answer away, so wrong ones are mixed
// in. Two things are therefore kept independent of correctness: how many
// double-accent options are offered, which is doubles either way, and where
// among them the correct one sits, which is random.
//
// A genuinely double-stressed word always offers at least one, since dropping
// it would leave the question unanswerable.
func AnswerVariants(word string, doubles int) ([]string, error) {
	correct, err := ParseAccentMask(word)
	if err != nil {
		return nil, err
	}

	variants, err := GetPossibleAccents(word)
	if err != nil {
		return nil, err
	}

	// Nothing in the approved list carries more than two stresses. If one ever
	// did, offer it as it is rather than guessing at decoys for it.
	if len(correct) > 2 {
		return append(variants, word), nil
	}

	correctIsDouble := len(correct) == 2
	if correctIsDouble && doubles < 1 {
		doubles = 1
	}

	if doubles <= 0 {
		return variants, nil
	}

	chosen := make([]string, 0, doubles)
	if correctIsDouble {
		chosen = append(chosen, word)
	}

	chosen = append(chosen, decoys(doubleVariants(word), word, doubles-len(chosen))...)

	rand.Shuffle(len(chosen), func(i, j int) { chosen[i], chosen[j] = chosen[j], chosen[i] })

	return append(variants, chosen...), nil
}

// decoys draws up to n double-accent variants at random, never the correct one.
func decoys(pool []string, correct string, n int) []string {
	if n <= 0 {
		return nil
	}

	wrong := make([]string, 0, len(pool))
	for _, variant := range pool {
		if variant != correct {
			wrong = append(wrong, variant)
		}
	}

	rand.Shuffle(len(wrong), func(i, j int) { wrong[i], wrong[j] = wrong[j], wrong[i] })

	return wrong[:min(n, len(wrong))]
}

// doubleVariants returns every way of stressing two of the word's vowels.
func doubleVariants(word string) []string {
	positions := vowelPositions(word)

	variants := make([]string, 0, len(positions)*(len(positions)-1)/2)
	for i := range positions {
		for j := i + 1; j < len(positions); j++ {
			variants = append(variants, applyAccentMask(word, AccentMask{positions[i], positions[j]}))
		}
	}

	return variants
}

// vowelPositions returns the rune indexes of the word's vowels.
func vowelPositions(word string) []int {
	var positions []int
	for i, r := range []rune(Normalize(word)) {
		if isVowel(r) {
			positions = append(positions, i)
		}
	}

	return positions
}

// HasDoubleStress reports whether a variant stresses more than one vowel.
func HasDoubleStress(variant string) bool {
	mask, err := ParseAccentMask(variant)

	return err == nil && len(mask) > 1
}
