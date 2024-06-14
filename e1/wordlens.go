package e1

import "sync"

type WordLens struct {
	mu sync.Mutex
}

func NewWordLens() WordLens {
	return WordLens{
		mu: sync.Mutex{},
	}
}

func (wl *WordLens) FindPalindromes(words []string, concurrently bool) map[string]int {
	palindromes := make(map[string]int)

	if concurrently {
		wg := sync.WaitGroup{}
		wg.Add(len(words))

		for _, word := range words {
			go func(word string) {
				defer wg.Done()
				if wl.isPalindrome(word) {
					wl.mu.Lock()
					palindromes[word]++
					wl.mu.Unlock()
				}
			}(word)
		}
		wg.Wait()
	} else {
		for _, word := range words {
			if wl.isPalindrome(word) {
				palindromes[word]++
			}
		}
	}

	return palindromes
}

func (wl *WordLens) isPalindrome(word string) bool {
	for i := range word {
		if word[i] != word[len(word)-1-i] {
			return false
		}
	}
	return true
}
