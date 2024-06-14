package e2_test

import (
	"flag"
	"strconv"
	"testing"

	"github.com/idiomat/goo11ynyt/benchmarking/wordlens"
	"github.com/idiomat/goo11ynyt/e2"
)

var technique = flag.String("technique", "sequential", "run the benchmark with the specified technique")

func TestFindPalindromes(t *testing.T) {
	tests := map[string]struct {
		words    []string
		expected int
	}{
		"empty": {
			words:    []string{},
			expected: 0,
		},
		"single": {
			words:    []string{"a"},
			expected: 1,
		},
		"multiple": {
			words:    []string{"racecar", "hello", "madam", "level", "deified", "rotator", "step on no pets", "not a palindrome"},
			expected: 6,
		},
		"lots": {
			words:    wordlens.TestWords(),
			expected: 27,
		},
	}

	wl := e2.NewWordLens()

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			res := wl.FindPalindromes(tc.words, false, e2.Technique("sequential"))
			if len(res) != tc.expected {
				t.Errorf("Expected %d palindromes, got %d", tc.expected, len(res))
			}
		})
	}
}

func BenchmarkFindPalindromes(b *testing.B) {
	b.StopTimer() // exclude preparations from the benchmark
	flag.Parse()
	wl := e2.NewWordLens()
	allWords := wordlens.TestWords()
	concurrent := *technique != "sequential"

	b.StartTimer() // run the benchmark
	for n := 25; n <= len(allWords); n = n + 25 {
		words := allWords[:n]
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				wl.FindPalindromes(words, concurrent, e2.Technique(*technique))
			}
		})
	}
}
