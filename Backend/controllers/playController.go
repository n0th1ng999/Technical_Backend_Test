package controllers

import (
	"errors"
	"main/config"
	"main/helpers"
	"main/middleware"
	"main/models"

	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// HandlePlay handles the play endpoint for betting
// It upgrades the HTTP to WebSocket Connection
// It checks if the player is authenticated Returns an error Message if So
// It checks if the player is playing (aka: if already connected to the server through a socket), If so it Returns an error
// If he wasn't playing change his playing status to true
// On connection Close Change his playing status to false
func HandlePlayWS(w http.ResponseWriter, r *http.Request) {

	// Try to upgrade to Websockets
	conn, upgradeConErr := helpers.WSUpgrader.Upgrade(w, r, nil)
	// On error
	if upgradeConErr != nil {
		return
	}

	// Get Player initial information
	player, authError := middleware.AuthenticateUser(r)
	if authError != nil {
		response := map[string]interface{}{"code": 401, "message": authError.Error()}
		stringifiedResponse, _ := helpers.JsonStringifier(response)

		conn.WriteMessage(websocket.TextMessage, []byte(stringifiedResponse))
		// Conn.WriteControl Should send a close code and a message but it's not working properly.
		conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "No auth token provided"), time.Now())
		conn.Close()
		return
	}

	// Timeout Duration based on ENV CONFIG
	timeoutDuration := time.Duration(config.SOCKET_TIMEOUT_DURATION * float32(time.Second))

	// Set Timeout
	timeout := time.AfterFunc(timeoutDuration, func() {
		conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "User is Inactive, Timeout Exceeded"), time.Now()) // Forcefully close the connection
		return
	})

	// Create a timeout function

	for {

		// Receive ws message
		messageType, receivedMsg, readMessageErr := conn.ReadMessage() // BLOCKING
		timeout.Stop()
		if readMessageErr != nil {
			if websocket.IsUnexpectedCloseError(readMessageErr, websocket.CloseGoingAway) {
				break
			}
			continue
		}

		// Default Response
		response := map[string]interface{}{
			"message": "Bet placed successfully",
			"code":    200,
		}

		// Error List
		errorList := []string{}

		parsedData, jsonParserErr := helpers.JsonParser(receivedMsg)
		if jsonParserErr != nil {
			errorList = append(errorList, "Invalid JSON received")

			response["errorsList"] = errorList
			response["code"] = 400
			response["message"] = "Invalid JSON"

			stringifiedResponse, _ := helpers.JsonStringifier(response)
			conn.WriteMessage(messageType, []byte(stringifiedResponse))
			continue // Skip the rest of the loop
		}

		// If no bet amount or type of bet is provided return an error

		// betAmount := float32(betAmount) // Transform from 64 to float32 If performance is critical

		// Verify is a String or exists and extract betType
		betType, betTypeIsString := parsedData["betType"].(string)
		if !betTypeIsString {
			errorList = append(errorList, "Invalid or missing betType")
		}

		// Validate betType
		if betType != "pair" && betType != "not pair" {
			errorList = append(errorList, "betType must be 'pair' or 'not pair'")
		}

		// Verify is a float32 or exists and extract betAmount
		betAmount64, betAmountIsFloat64 := parsedData["betAmount"].(float64) // Assuming betAmount is of type float64
		if !betAmountIsFloat64 {
			errorList = append(errorList, "Invalid or missing betAmount")
		}

		// Convert float64 to float32
		betAmount32 := float32(betAmount64)

		// Check if it's greater than 0
		if betAmount32 <= 0 {
			errorList = append(errorList, "betAmount must be greater than 0")
		}

		// Check if Player Is Already Betting
		isAlreadyBetting, errCheckingStatus := models.CheckPlayerBettingStatus(player.ID)
		if errCheckingStatus != nil {
			errorList = append(errorList, errCheckingStatus.Error())
		}

		// Check if Player Is Already Betting
		if isAlreadyBetting {

			errorList = append(errorList, "Player already betting, please await the bet processing...")
		}

		// Process Betting if player is not already in Betting Process and if the status was properly provided
		if !isAlreadyBetting && errCheckingStatus == nil && len(errorList) == 0 {

			models.UpdatePlayerBettingStatus(player.ID, true)

			diceRollResult, err := processBet(player.ID, betAmount32, betType)
			if err != nil {
				errorList = append(errorList, err.Error())
			} else {
				response["DiceNumber"] = diceRollResult.DiceNumber
				response["PlayerWin"] = diceRollResult.PlayerWin
				response["PlayerOriginalBet"] = diceRollResult.PlayerOriginalBet
				response["PlayerMessage"] = diceRollResult.PlayerMessage
				response["Winnings"] = diceRollResult.Winnings
			}

			models.UpdatePlayerBettingStatus(player.ID, false)
		}

		// Check if any errors occurred
		if len(errorList) > 0 {
			response["errorsList"] = errorList
			response["code"] = 400
			response["message"] = "Error creating bet, check error list"
		}

		// Send the stringified response JSON
		stringifiedResponse, _ := helpers.JsonStringifier(response)

		if writingMessageErr := conn.WriteMessage(messageType, []byte(stringifiedResponse)); writingMessageErr != nil {

			break
		}

		// Timeout reset
		timeout = time.AfterFunc(timeoutDuration, func() {
			conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "User is Inactive, Timeout Exceeded"), time.Now()) // Forcefully close the connection
			return
		})

	}
	conn.Close()
}

type DiceRollResult struct {
	DiceNumber        int
	PlayerWin         bool
	PlayerOriginalBet string
	PlayerMessage     string
	Winnings          float32 // Winnings of the player on the bet (either be it negative or positive)
}

// Return betResult, Number of dice, and the type (pair / not pair)
func processBet(playerId int, betAmount float32, betType string) (DiceRollResult, error) {
	// Get Current Info on Player
	player, err := models.GetPlayerByID(playerId)
	if err != nil {
		return DiceRollResult{}, errors.New("Error getting player info for bet processing")
	}

	// Check if Bet Amount is Above BetBalance + Wallet
	if betAmount > (player.BetBalance + player.Wallet) {
		return DiceRollResult{}, errors.New("betAmount exceeds player's balance and bet balance")
	} else {
		player.DeductBetAmount(betAmount) // Subtract from Wallet
		// ( I've Only remembered now that structs can have functions  RIP )
	}

	// PS: This is here only to demonstrate that during a dice roll the player cannot bet again
	start := time.Now() // Starts a counter

	diceRollEnd := false

	RolledDiceNumber := rand.Intn(6) + 1

	if config.RIGGED_DICE_NUMBER != 0 { // Default value aka not rigged
		RolledDiceNumber = config.RIGGED_DICE_NUMBER
	}

	diceRollResult := DiceRollResult{
		DiceNumber:        RolledDiceNumber, // Resulting Dice Number
		PlayerOriginalBet: betType,
	}

	// Check if the player won
	playerWon := (RolledDiceNumber%2 == 0 && betType == "pair") || (RolledDiceNumber%2 != 0 && betType == "not pair")

	// If player wins the bet
	if playerWon {
		player.BetBalance += betAmount * config.WINNING_MULTIPLIER // MULTIPLIER FROM ENV
		diceRollResult.PlayerMessage = "You've Won :)"
		diceRollResult.Winnings = betAmount * config.WINNING_MULTIPLIER
		diceRollResult.PlayerWin = true

	} else {
		diceRollResult.PlayerMessage = "You've Lost :("
		diceRollResult.Winnings = -betAmount
		diceRollResult.PlayerWin = false
	}

	// Update Player's Balance and Wallet
	updateBalanceError := models.UpdatePlayerBalance(player.ID, player.Wallet, player.BetBalance)
	if updateBalanceError != nil {
		return DiceRollResult{}, updateBalanceError
	}

	diceRollEnd = true
	for {
		// Loop until 2 seconds have passed
		if time.Since(start).Seconds() >= float64(config.PROCESSING_DURATION) && diceRollEnd { // Bet processing minimum Time
			break
		}

	}

	return diceRollResult, nil
}
