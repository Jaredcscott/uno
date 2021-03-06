package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/jak103/uno/db"
	"github.com/jak103/uno/model"
)

////////////////////////////////////////////////////////////
// These are all of the functions for the game -> essentially public functions
////////////////////////////////////////////////////////////
func getGameUpdate(gameID string, playerID string) (*model.Game, error) {
	database, err := db.GetDb()

	if err != nil {
		return nil, err
	}

	gameData, gameErr := database.LookupGameByID(gameID)
	if gameErr != nil {
		return nil, err
	}

	// Determine if player is active
	now := time.Now()
	changedData := false
	for index, player := range gameData.Players {
		if player.ID == playerID {
			gameData.Players[index].LastUpdated = now.Format(time.RFC3339)
			gameData.Players[index].IsActive = true
			changedData = true
		} else if player.IsActive && player.LastUpdated != "" {
			lastUpdated, _ := time.Parse(time.RFC3339, player.LastUpdated)
			adjustedTime := lastUpdated.Add(time.Second * 10)
			if adjustedTime.Before(now) {
				gameData.Players[index].IsActive = false
				changedData = true
			}
		}
	}

	if changedData {
		gameErr = database.SaveGame(*gameData)
		if gameErr != nil {
			return nil, err
		}
	}

	return gameData, nil
}

func createPlayer(name string) (*model.Player, error) {
	database, err := db.GetDb()
	if err != nil {
		return nil, err
	}

	player, err := database.CreatePlayer(name)
	if err != nil {
		return nil, err
	}

	return player, nil
}

func createNewGame(gameName string, creatorName string) (*model.Game, *model.Player, error) {
	database, err := db.GetDb()
	if err != nil {
		return nil, nil, err
	}

	creator, err := database.CreatePlayer(creatorName)
	if err != nil {
		return nil, nil, err
	}

	game, err := database.CreateGame(gameName, creator.ID)
	if err != nil {
		return nil, nil, err
	}

	err = database.SaveGame(*game)
	if err != nil {
		return nil, nil, err
	}

	return game, creator, nil
}

func joinGame(game string, player *model.Player) (*model.Game, error) {
	database, err := db.GetDb()
	if err != nil {
		return nil, err
	}

	gameData, gameErr := database.JoinGame(game, player.ID)

	if gameErr != nil {
		return nil, gameErr
	}

	return gameData, nil
}

func addMessage(gameID string, playerID string, message model.Message) (*model.Game, error) { //*model.Player
	database, err := db.GetDb()

	if err != nil {
		return nil, err
	}

	gameData, err := database.AddMessage(gameID, playerID, message)

	if err != nil {
		return nil, err
	}

	err = database.SaveGame(*gameData)

	if err != nil {
		return nil, err
	}

	return gameData, nil
}

func playCard(game string, playerID string, card model.Card) (*model.Game, error) {
	database, err := db.GetDb()

	if err != nil {
		return nil, err
	}

	gameData, gameErr := database.LookupGameByID(game)

	if gameErr != nil {
		return nil, err
	}

	if gameData.Players[gameData.CurrentPlayer].ID == playerID {
		hand := gameData.Players[gameData.CurrentPlayer].Cards
		if checkForCardInHand(card, hand) && isCardPlayable(card, gameData.DiscardPile) {
			// Valid card can be played

			gameData.DiscardPile = append(gameData.DiscardPile, card)

			for index, item := range hand {
				if item == card || (item.Value == "W" && card.Value == "W") || (item.Value == "W4" && card.Value == "W4") {
					gameData.Players[gameData.CurrentPlayer].Cards = append(hand[:index], hand[index+1:]...)
					break
				}
			}

			// Reset Uno calling protection after every card is played
			gameData.Players[gameData.CurrentPlayer].Protection = false

			// Update who plays next, taking into account reverse card and skip card
			if card.Value == "R" {
				gameData.Direction = !gameData.Direction
				if len(gameData.Players) == 2 {
					gameData = goToNextPlayer(gameData)
				}
			}

			if card.Value == "S" {
				gameData = goToNextPlayer(gameData)
			}

			gameData = goToNextPlayer(gameData)

			// take into account cards that force the next player to draw
			if card.Value == "D2" {
				gameData = drawNCards(gameData, 2)
				gameData = goToNextPlayer(gameData)
			}

			if card.Value == "W4" {
				gameData = drawNCards(gameData, 4)
				gameData = goToNextPlayer(gameData)
			}
		}
	}

	err = database.SaveGame(*gameData)

	if err != nil {
		return nil, err
	}

	return gameData, nil
}

func logicCallUno(gameID string, callingPlayerID string, calledOnPlayerID string) (*model.Game, error) {
	database, dbErr := db.GetDb()

	if dbErr != nil {
		return nil, dbErr
	}

	gameData, gameErr := database.LookupGameByID(gameID)

	if gameErr != nil {
		return nil, gameErr
	}

	var callingPlayer, calledOnPlayer *model.Player
	for i := range gameData.Players {
		if gameData.Players[i].ID == callingPlayerID {
			callingPlayer = &gameData.Players[i]
		}

		if gameData.Players[i].ID == calledOnPlayerID {
			calledOnPlayer = &gameData.Players[i]
		}
	}

	var drawnCard model.Card
	if len(calledOnPlayer.Cards) == 1 {
		if calledOnPlayer.Protection == false {
			if calledOnPlayer.ID == callingPlayer.ID {
				calledOnPlayer.Protection = true
			} else {
				for i := 0; i < 4; i++ {
					gameData, drawnCard = drawTopCard(gameData)

					calledOnPlayer.Cards = append(calledOnPlayer.Cards, drawnCard)
				}
			}
		}
	} else {
		gameData, drawnCard = drawTopCard(gameData)

		callingPlayer.Cards = append(callingPlayer.Cards, drawnCard)
	}

	database.SaveGame(*gameData)

	return gameData, nil
}

func drawCard(gameID string, playerID string) (*model.Game, error) {
	// These lines are simply getting the database and game and handling any error that could occur
	database, dbErr := db.GetDb()

	if dbErr != nil {
		return nil, dbErr
	}

	gameData, gameErr := database.LookupGameByID(gameID)

	if gameErr != nil {
		return nil, gameErr
	}

	// Reset Uno calling protection after a card is drawn
	gameData.Players[gameData.CurrentPlayer].Protection = false

	// We get the current player from the game
	player := &gameData.Players[gameData.CurrentPlayer]
	//We then check if the player attempting to play a card is the current player
	if player.ID == playerID {
		// We check if the draw pile has available cards
		if len(gameData.DrawPile) == 0 {
			// we check that the discard pile has cards to reshuffle
			if len(gameData.DiscardPile) <= 1 {
				// If there are not cards on the table add a new deck
				// TODO in the future do more complicated logic such as skip the players turn or something like that.
				gameData.DrawPile = generateShuffledDeck(1)
			} else {
				gameData = reshuffleDiscardPile(gameData)
			}
		}

		// Draw a card off the drawpile
		_, drawnCard := drawTopCard(gameData)

		// append the card into the players cards from the draw pile
		player.Cards = append(player.Cards, drawnCard)

		// if the card cannot be played, advance to the next player
		if !isCardPlayable(drawnCard, gameData.DiscardPile) {
			gameData = goToNextPlayer(gameData)
		}

		// Save the game into the database
		database.SaveGame(*gameData)

		// Return a successfully updated game.
		return gameData, nil
	}

	// Check why they couldn't draw, is it not their turn, or are they not part of this game?
	for _, item := range gameData.Players {
		if item.ID == playerID {
			return nil, fmt.Errorf("It is not your turn to play")
		}
	}

	// TODO Make real error
	return nil, fmt.Errorf("You cannot participate in a game you do not belong")

}

/*This function will:
Deal out 7 cards to each player
Set the first card for the game to start from
*/
func dealCards(game *model.Game) (*model.Game, error) {

	// pick a starting player
	game.CurrentPlayer = rand.Intn(len(game.Players))

	// get a deck
	game.DrawPile = generateShuffledDeck(len(game.Players))

	//For each player currently in the game, give everyone 7 cards
	for k := range game.Players {
		cards := []model.Card{}
		for i := 0; i < 7; i++ {

			var drawnCard model.Card
			game, drawnCard = drawTopCard(game)
			cards = append(cards, drawnCard)

		}
		//Add all 7 cards to that players hand
		game.Players[k].Cards = cards
	}
	// draw a card for the discard
	var drawnCard model.Card
	game, drawnCard = drawTopCard(game)

	// ensure that this first card is a number card
	for !isNumberCard(drawnCard) {
		// if not, add it back to the draw pile
		game.DrawPile = append(game.DrawPile, drawnCard)
		// reshuffle cards so the same card is not drawn again
		game.DrawPile = shuffleCards(game.DrawPile)
		// draw a new card
		game, drawnCard = drawTopCard(game)
	}

	game.DiscardPile = append(game.DiscardPile, drawnCard)

	game.Status = "Playing"

	database, err := db.GetDb()

	if err != nil {
		return nil, err
	}

	// save the new game status
	err = database.SaveGame(*game)

	return game, err
}

////////////////////////////////////////////////////////////
// Utility Functions
////////////////////////////////////////////////////////////

// Checks if a card is playable
// Card is playable if it is wild or matches the card on top of the discard pile
// Does not check that the card is in the player's hand
// Use checkForCardInHand for that
func isCardPlayable(card model.Card, discardPile []model.Card) bool {
	isWild := card.Value == "W4" || card.Value == "W"

	cardOnDiscardPile := discardPile[len(discardPile)-1]

	if (card.Color == cardOnDiscardPile.Color || card.Value == cardOnDiscardPile.Value || isWild) {
		return true
	}

	return false
}

func checkForCardInHand(card model.Card, hand []model.Card) bool {
	for _, c := range hand {
		// the wild cards, W4 and W, don't need to match in color; not for the previous card, and not with the hand. The card itself can become any color.
		if c.Value == card.Value && (c.Color == card.Color || card.Value == "W4" || card.Value == "W") {
			return true
		}
	}
	return false
}

func goToNextPlayer(gameData *model.Game) *model.Game {
	//check for winner
	if len(gameData.Players[gameData.CurrentPlayer].Cards) == 0 {
		gameData.GameOver = gameData.Players[gameData.CurrentPlayer].Name
		gameData.Status = model.Finished
	} else {
		if gameData.Direction {
			gameData.CurrentPlayer++
			gameData.CurrentPlayer %= len(gameData.Players)
		} else {
			gameData.CurrentPlayer--
			if gameData.CurrentPlayer < 0 {
				gameData.CurrentPlayer = len(gameData.Players) - 1
			}
		}
	}

	return gameData
}

func reshuffleDiscardPile(gameData *model.Game) *model.Game {
	//Reshuffle all discarded cards except the last one back into the draw pile.
	oldDiscard := gameData.DiscardPile[:len(gameData.DiscardPile)-1]
	gameData.DrawPile = shuffleCards(oldDiscard)
	gameData.DiscardPile = gameData.DiscardPile[len(gameData.DiscardPile)-1:]
	return gameData
}

func drawNCards(gameData *model.Game, nCards uint) *model.Game {
	for i := uint(0); i < nCards; i++ {
		var drawnCard model.Card
		gameData, drawnCard = drawTopCard(gameData)
		gameData.Players[gameData.CurrentPlayer].Cards = append(gameData.Players[gameData.CurrentPlayer].Cards, drawnCard)
	}
	return gameData
}

func drawTopCard(game *model.Game) (*model.Game, model.Card) {
	drawnCard := game.DrawPile[len(game.DrawPile)-1]
	game.DrawPile = game.DrawPile[:len(game.DrawPile)-1]
	return game, drawnCard
}

func checkGameExists(gameID string) (bool, error) {
	database, err := db.GetDb()

	if err != nil {
		return false, err
	}

	_, gameErr := database.LookupGameByID(gameID)

	if gameErr != nil {
		return false, gameErr
	}

	return true, nil
}
