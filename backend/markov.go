package main

import (
	"math/rand"
	"strings"
	"sync"
	"unicode"
)

// MarkovChain is a bigram (2-gram) model: given the last word, predict the next.
// We store trigram context too (last 2 words → next word) for better quality.
type MarkovChain struct {
	mu    sync.RWMutex
	// bigrams: word → []nextWord
	bigrams  map[string][]string
	// trigrams: "w1 w2" → []nextWord
	trigrams map[string][]string
}

var globalMarkov = &MarkovChain{
	bigrams:  make(map[string][]string),
	trigrams: make(map[string][]string),
}

// seedMarkdownCorpus trains the chain on common markdown patterns
func init() {
	corpus := `
# Introduction
## Overview
### Summary
#### Details
##### Notes

The document contains the following sections.
This section describes the main features of the system.
The following list shows the available options.
Please refer to the documentation for more details.

- First item in the list
- Second item with description
- Third item and more details
- Another important point to consider

1. First step in the process
2. Second step to follow
3. Third step and final

**Bold text** and *italic text* are supported.
You can also use ~~strikethrough~~ formatting.
Inline code like this is also supported.

> This is a blockquote with important information.
> It can span multiple lines for longer quotes.

The main purpose of this tool is to enable collaboration.
Users can edit documents simultaneously in real time.
All changes are synchronized across connected clients.
The system uses WebSockets for low-latency communication.

For more information visit the documentation page.
Contact support if you need additional help.
See also the related articles and tutorials.

## Getting Started

To begin using the editor you need to create a document.
Share the document URL with your collaborators.
Everyone with the link can edit the document together.

## Features

Real-time synchronization keeps everyone in sync.
Markdown preview renders your content instantly.
The AI assistant suggests completions as you type.
Document history lets you review previous versions.

## API Reference

The WebSocket endpoint accepts JSON messages.
Each message must include a type and document ID.
The server broadcasts updates to all connected users.
Version numbers help resolve editing conflicts.
`
	globalMarkov.Train(corpus)
}

// Train updates the model with the given text corpus
func (m *MarkovChain) Train(text string) {
	words := tokenize(text)
	if len(words) < 2 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for i := 0; i < len(words)-1; i++ {
		w1 := words[i]
		w2 := words[i+1]
		m.bigrams[w1] = append(m.bigrams[w1], w2)

		if i < len(words)-2 {
			key := w1 + " " + w2
			w3 := words[i+2]
			m.trigrams[key] = append(m.trigrams[key], w3)
		}
	}
}

// Suggest returns up to n word suggestions given a text prefix.
// It uses the last 1-2 words as context.
func (m *MarkovChain) Suggest(prefix string, n int) []string {
	words := tokenize(prefix)
	if len(words) == 0 {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try trigram first
	if len(words) >= 2 {
		key := words[len(words)-2] + " " + words[len(words)-1]
		if nexts, ok := m.trigrams[key]; ok && len(nexts) > 0 {
			return pickUnique(nexts, n)
		}
	}

	// Fall back to bigram
	last := words[len(words)-1]
	if nexts, ok := m.bigrams[last]; ok && len(nexts) > 0 {
		return pickUnique(nexts, n)
	}

	// Fall back to bigram on lowercased last word
	last = strings.ToLower(last)
	if nexts, ok := m.bigrams[last]; ok && len(nexts) > 0 {
		return pickUnique(nexts, n)
	}

	return nil
}

// tokenize splits text into lowercase words, stripping markdown syntax
func tokenize(text string) []string {
	// Remove markdown symbols but keep words
	clean := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '\n' {
			return r
		}
		return ' '
	}, text)

	parts := strings.Fields(strings.ToLower(clean))
	var out []string
	for _, p := range parts {
		if len(p) > 1 {
			out = append(out, p)
		}
	}
	return out
}

func pickUnique(words []string, n int) []string {
	// Shuffle a copy to get variety
	cp := make([]string, len(words))
	copy(cp, words)
	rand.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })

	seen := make(map[string]bool)
	var result []string
	for _, w := range cp {
		if !seen[w] {
			seen[w] = true
			result = append(result, w)
		}
		if len(result) >= n {
			break
		}
	}
	return result
}
