package main

import (
	"strings"
	"unicode"
)

func similarPlans(content string) ([]plan, error) {
	all, err := plans()
	if err != nil {
		return nil, err
	}
	var matches []plan
	for _, candidate := range all {
		if similarText(content, candidate.Content) {
			matches = append(matches, candidate)
		}
	}
	return matches, nil
}

func similarText(first, second string) bool {
	first, second = normalizeText(first), normalizeText(second)
	if first == second || strings.Contains(first, second) || strings.Contains(second, first) {
		return true
	}
	firstWords, secondWords := wordSet(first), wordSet(second)
	if len(firstWords) == 0 || len(secondWords) == 0 {
		return false
	}
	shared := 0
	for word := range firstWords {
		if secondWords[word] {
			shared++
		}
	}
	minimum := min(len(firstWords), len(secondWords))
	return float64(shared)/float64(minimum) >= 0.7
}

func normalizeText(value string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }), " ")
}

func wordSet(value string) map[string]bool {
	ignored := map[string]bool{"a": true, "an": true, "the": true, "my": true, "to": true, "for": true, "of": true, "and": true}
	words := make(map[string]bool)
	for _, word := range strings.Fields(value) {
		if !ignored[word] {
			words[word] = true
		}
	}
	return words
}
