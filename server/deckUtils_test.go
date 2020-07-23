package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestGenerateShuffledDeck(t *testing.T) {
	deck := generateShuffledDeck()

	// check that the deck has the right number of each color
	colorCounts := map[string]int {
		"red": 0,
		"blue": 0,
		"green": 0,
		"yellow": 0,
		"wild": 0,
	}

	for _, card := range deck {
		colorCounts[card.Color]++
	}
	
	assert.Equal(t, 25, colorCounts["red"])
	assert.Equal(t, 25, colorCounts["blue"])
	assert.Equal(t, 25, colorCounts["green"])
	assert.Equal(t, 25, colorCounts["yellow"])
	assert.Equal(t, 8, colorCounts["wild"])

	// check that the deck has the right number of total cards
	assert.Equal(t, 108, len(deck))
}

func TestShuffleCards(t *testing.T) { 
	deck := shuffleCards([]UnoCard{UnoCard{"red", "1"},UnoCard{"blue", "2"},UnoCard{"green", "3"}})
	assert.NotEqual(t, deck[:0], UnoCard{"red", "1"})
}

func TestPrintCard(t *testing.T) { 
	card := UnoCard{"red", "1"}
	printCard(card)
	//output //This needs to be the 'output' of the print card function
	//assert.Equal(t,"red 1",output)
}