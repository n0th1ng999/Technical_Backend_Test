package controllers

import (
	"errors"
	"main/config"
	"main/helpers"
	"main/middleware"
	"main/models"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

/*
If player is not currently betting then the endpoint will transfer the bettingBalance to Wallet
Setting isBetting to true to prevents bets being made at the same time or other endPlay Requests
A Player Requests for EndPlay Endpoint must have the following body : {"cashInAmount": float64}

! If a player requests for a cashIn <= 0 and more then their betBalance -> returns an error
* If a player requests for a cashIn >= 0 and less then their betBalance -> returns success
? Within the UpdatePlayerBalance function an event is emitted to the wallet controller
? that is subscribed via a go routine
*/
func HandleEndPlayWS(w http.ResponseWriter, r *http.Request) {
	conn, upgradeConErr := helpers.WSUpgrader.Upgrade(w, r, nil)
	// On error
	if upgradeConErr != nil {
		return
	}

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
	// Timeout float to seconds
	timeoutDuration := time.Duration(config.SOCKET_TIMEOUT_DURATION * float32(time.Second))

	// Set Timeout
	timeout := time.AfterFunc(timeoutDuration, func() {
		conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "User is Inactive, Timeout Exceeded"), time.Now()) // Forcefully close the connection
		return
	})

	for {

		// Receive ws message
		messageType, receivedMsg, readMessageErr := conn.ReadMessage() // Blocking
		if readMessageErr != nil {
			if websocket.IsUnexpectedCloseError(readMessageErr, websocket.CloseGoingAway) {
				break
			}
			continue
		}

		//
		timeout.Stop()

		// Default Response
		response := map[string]interface{}{
			"message": "Cash In Successful",
			"code":    200,
		}

		// Error List
		errorList := []string{}

		parsedData, readMessageErr := helpers.JsonParser(receivedMsg)
		if readMessageErr != nil {
			errorList = append(errorList, "Invalid JSON received")

			response["errorsList"] = errorList
			response["code"] = 400
			response["message"] = "Invalid JSON"

			stringifiedResponse, _ := helpers.JsonStringifier(response)
			conn.WriteMessage(messageType, []byte(stringifiedResponse))
			continue // Skip the rest of the loop
		}

		cashInAmount64, cashInAmountIsFloat64 := parsedData["cashInAmount"].(float64)
		if !cashInAmountIsFloat64 {
			errorList = append(errorList, "Invalid cashInAmount Type")
			response["errorsList"] = errorList
			response["code"] = 400
		}

		// Convert float64 to float32
		cashInAmount32 := float32(cashInAmount64)

		if cashInAmount32 <= 0 {
			errorList = append(errorList, "Invalid cashInAmount, it must be more than zero")
			response["errorsList"] = errorList
			response["code"] = 400

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
		if !isAlreadyBetting && errCheckingStatus == nil {

			models.UpdatePlayerBettingStatus(player.ID, true)

			err := processCashIn(player.ID, cashInAmount32)
			if err != nil {
				errorList = append(errorList, err.Error())

			}

			models.UpdatePlayerBettingStatus(player.ID, false)
		}

		// Check if any errors occurred
		if len(errorList) > 0 {
			response["errorsList"] = errorList
			response["code"] = 400
			response["message"] = "Error cashing bet balance, check error list"
		}

		stringifiedResponse, _ := helpers.JsonStringifier(response)

		if writingMessageErr := conn.WriteMessage(messageType, []byte(stringifiedResponse)); writingMessageErr != nil {
			break
		}

		// Timeout Reset
		timeout = time.AfterFunc(timeoutDuration, func() {
			conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "User is Inactive, Timeout Exceeded"), time.Now()) // Forcefully close the connection
			return
		})

	}

	conn.Close()

}

func processCashIn(id int, cashInAmount float32) error {

	start := time.Now()
	isProcessed := false

	// Find Current Player Info By ID
	player, errFindingPlayer := models.GetPlayerByID(id)
	if errFindingPlayer != nil {
		return errFindingPlayer
	}

	// Check if Player Has Enough Bet Balance
	if player.BetBalance < cashInAmount {
		return errors.New("Player does not have enough bet Balance to cash In")
	}

	// Transfer the betting balance to Wallet
	updatingBalanceError := models.UpdatePlayerBalance(id, player.Wallet+cashInAmount, player.BetBalance-cashInAmount)
	if updatingBalanceError != nil {
		return updatingBalanceError
	}

	isProcessed = true

	for {
		// Loop until 2 seconds have passed
		if time.Since(start).Seconds() >= float64(config.PROCESSING_DURATION) && isProcessed { // Bet processing minimum Time
			break
		}

	}

	return nil
}
