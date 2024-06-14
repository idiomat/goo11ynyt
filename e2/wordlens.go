package e2

import (
	"runtime"
	"sync"
)

type WordLens struct {
	mu      sync.Mutex
	workers int
}

func NewWordLens() WordLens {
	return WordLens{
		mu:      sync.Mutex{},
		workers: runtime.NumCPU(),
	}
}

type Technique string

const (
	TechniqueMutex   Technique = "mutex"
	TechniqueChannel Technique = "channel"
	TechniqueWorkers Technique = "workers"
)

func (wl *WordLens) FindPalindromes(words []string, useConcurrency bool, tech Technique) map[string]int {
	palindromes := make(map[string]int)

	switch tech {
	case TechniqueMutex:
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
	case TechniqueChannel:
		wg := sync.WaitGroup{}
		wg.Add(len(words))
		results := make(chan string, len(words))

		for _, word := range words {
			go func(word string) {
				defer wg.Done()
				if wl.isPalindrome(word) {
					results <- word
				}
			}(word)
		}

		go func() {
			wg.Wait()
			close(results)
		}()

		for word := range results {
			palindromes[word]++
		}
	case TechniqueWorkers:
		jobs := make(chan string, len(words))
		results := make(chan string)

		// start workers
		for w := 0; w < wl.workers; w++ {
			go wl.worker(jobs, results)
		}

		// start jobs
		go func() {
			for _, word := range words {
				jobs <- word
			}
			close(jobs)
		}()

		// collect results
		for i := 0; i < len(words); i++ {
			if word := <-results; word != "" {
				palindromes[word]++
			}
		}
		close(results)
	default:
		for _, word := range words {
			if wl.isPalindrome(word) {
				palindromes[word]++
			}
		}
	}
	return palindromes
}

func (wl *WordLens) worker(jobs <-chan string, results chan<- string) {
	for word := range jobs {
		if wl.isPalindrome(word) {
			results <- word
		} else {
			results <- ""
		}
	}
}

func (wl *WordLens) isPalindrome(word string) bool {
	for i := range word {
		if word[i] != word[len(word)-1-i] {
			return false
		}
	}
	return true
}
