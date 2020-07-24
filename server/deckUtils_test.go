package main

import (
	"testing"
	"github.com/stretchr/testify/assert"

	"bytes"
	"io"
	"log"
	"os"
	"sync"
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
	re := captureOutput(func() {
		printCard(card)
	})
	assert.Equal(t,"red 1\n",re)
}

func TestPrintCards(t *testing.T) { 
	deck := []UnoCard{UnoCard{"red", "1"}, UnoCard{"blue", "2"}, UnoCard{"green", "3"}}
	re := captureOutput(func() {
		printCards(deck)
	})
	assert.Equal(t,"red 1\nblue 2\ngreen 3\n",re)
}

func captureOutput(f func()) string {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout //Storing the reference for reassinging
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
		log.SetOutput(os.Stderr)
	}()
	os.Stdout = writer
	os.Stderr = writer
	log.SetOutput(writer)
	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		var buf bytes.Buffer
		wg.Done()
		io.Copy(&buf, reader)
		out <- buf.String()
	}()
	wg.Wait()
	f()
	writer.Close()
	return <-out
}
