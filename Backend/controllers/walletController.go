package controllers

import (
	"encoding/json"
	"fmt"
	"main/config"
	"main/events"
	"main/helpers"
	"main/middleware"
	"main/models"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func HandleWalletWS(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSockets
	conn, upgradeConErr := helpers.WSUpgrader.Upgrade(w, r, nil)
	if upgradeConErr != nil {
		return
	}

	// Authenticate User
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

	// Prepare and send a welcome message
	response := map[string]interface{}{
		"message":    "Wallet and bet balance retrieved with success!",
		"status":     "success",
		"wallet":     player.Wallet,
		"betBalance": player.BetBalance,
	}

	// Error List
	errorList := []string{}

	// On JSON format error
	stringifiedResponse, strError := helpers.JsonStringifier(response)
	if strError != nil {
		errorList = append(errorList, "Invalid JSON received")
		response["errorsList"] = errorList
		response["code"] = 400
		response["message"] = "Error! check error list."
		return
	}

	if writeMessageErr := conn.WriteMessage(websocket.TextMessage, []byte(stringifiedResponse)); writeMessageErr != nil {

		return
	}

	// Register Listener for Balance Update Events (once)
	balanceUpdateEvent := fmt.Sprintf("BalanceUpdate_%d", player.ID)
	handlerId := events.GlobalEmitter.On(balanceUpdateEvent, func(balanceData events.EventWalletData) {

		response["code"] = 200
		response["message"] = "Wallet / BetBalance Updated"
		response["wallet"] = balanceData.Wallet
		response["betBalance"] = balanceData.BetBalance

		// Convert Data to JSON and Send to Client
		stringifiedResponse, _ := helpers.JsonStringifier(response)

		if writeMessageErr := conn.WriteMessage(websocket.TextMessage, []byte(stringifiedResponse)); writeMessageErr != nil {

		}
	})

	// To keep connection alive (you can set a timeout here if necessary)
	time.Sleep(time.Duration(config.SOCKET_TIMEOUT_DURATION) * time.Second)

	// Unsubscribe to prevent memory issues
	events.GlobalEmitter.Off(balanceUpdateEvent, handlerId)

	// Close WebSocket Connection
	conn.Close()
}

type DepositReqBody struct {
	AmountToDeposit float32 `json:"amountToDeposit"`
}

func HandleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	player, err := middleware.AuthenticateUser(r)
	if err != nil {
		response := map[string]interface{}{
			"message": err.Error(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	var depositReqBody DepositReqBody
	err = json.NewDecoder(r.Body).Decode(&depositReqBody)
	if err != nil {
		response := map[string]interface{}{
			"message": "Request is missing data (amountToDeposit)",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if depositReqBody.AmountToDeposit <= 0 {
		response := map[string]interface{}{
			"message": "Deposit must be above 0",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return

	}

	if depositReqBody.AmountToDeposit > 1000000 {
		response := map[string]interface{}{
			"message": "Deposit must be below or equal to 1.000.000 (you can't be that rich!)",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return

	}

	isBetting, checkPlayerBettingStatusErr := models.CheckPlayerBettingStatus(player.ID)
	if checkPlayerBettingStatusErr != nil {
		response := map[string]interface{}{
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if isBetting {
		response := map[string]interface{}{
			"message": "Cannot deposit while player is in Betting Process",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	updateBettingStatusError := models.UpdatePlayerBettingStatus(player.ID, true)
	if updateBettingStatusError != nil {
		response := map[string]interface{}{
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Start timer to test race conditions easier
	start := time.Now()
	isProcessed := false

	player, err = models.GetPlayerByID(player.ID)
	if err != nil {
		response := map[string]interface{}{
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	err = models.UpdatePlayerBalance(player.ID, player.Wallet+depositReqBody.AmountToDeposit, player.BetBalance)
	if err != nil {
		response := map[string]interface{}{
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	updateBettingStatusError = models.UpdatePlayerBettingStatus(player.ID, false)
	if updateBettingStatusError != nil {
		response := map[string]interface{}{
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	isProcessed = true
	for {
		// Loop until 2 seconds have passed
		if time.Since(start).Seconds() >= float64(config.PROCESSING_DURATION) && isProcessed { // Bet processing minimum Time
			break
		}

	}

	// Handle deposit logic (e.g., update balance, save to DB, etc.)
	// For now, just send a success response
	response := map[string]interface{}{
		"message":    "Deposit successful",
		"amount":     depositReqBody.AmountToDeposit,
		"newBalance": player.Wallet + depositReqBody.AmountToDeposit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type WithdrawReqBody struct {
	AmountToWithdraw float32 `json:"amountToWithdraw"`
}

func HandleWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		return
	}

	// Authenticate user
	player, err := middleware.AuthenticateUser(r)
	if err != nil {
		response := map[string]interface{}{
			"message": err.Error(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	var withdrawReqBody WithdrawReqBody
	err = json.NewDecoder(r.Body).Decode(&withdrawReqBody)
	if err != nil {
		response := map[string]interface{}{
			"message": "Request is missing data (amountToWithdraw)",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validate that withdrawal amount is positive
	if withdrawReqBody.AmountToWithdraw <= 0 {
		response := map[string]interface{}{
			"message": "Withdraw amount must be greater than 0",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if player is already in the betting process
	isBetting, checkPlayerBettingStatusErr := models.CheckPlayerBettingStatus(player.ID)
	if checkPlayerBettingStatusErr != nil {
		response := map[string]interface{}{
			"message": checkPlayerBettingStatusErr.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if isBetting {
		response := map[string]interface{}{
			"message": "Cannot withdraw while player is in Betting Process",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update player's betting status to true to prevent racing updates during the withdraw process
	updateBettingStatusError := models.UpdatePlayerBettingStatus(player.ID, true) // Lock in processing
	if updateBettingStatusError != nil {
		response := map[string]interface{}{
			"message": updateBettingStatusError.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	// Start Timer to test race conditions easier
	start := time.Now()
	isProcessed := false

	// Fetch latest player details (to get their current wallet balance)
	player, err = models.GetPlayerByID(player.ID)
	if err != nil {
		response := map[string]interface{}{
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if the player has sufficient balance for the withdrawal
	if player.Wallet < withdrawReqBody.AmountToWithdraw {
		response := map[string]interface{}{
			"message": "Insufficient funds for withdrawal",
		}

		// Unlock the processing
		updateBettingStatusError = models.UpdatePlayerBettingStatus(player.ID, false)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update player's balance after withdrawal
	err = models.UpdatePlayerBalance(player.ID, player.Wallet-withdrawReqBody.AmountToWithdraw, player.BetBalance)
	if err != nil {
		response := map[string]interface{}{
			"message": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update betting status back to false
	updateBettingStatusError = models.UpdatePlayerBettingStatus(player.ID, false)
	if updateBettingStatusError != nil {
		response := map[string]interface{}{
			"message": updateBettingStatusError.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	isProcessed = true

	// End timer
	for {
		if time.Since(start).Seconds() >= float64(config.PROCESSING_DURATION) && isProcessed { // Bet processing minimum Time
			break
		}
	}

	// Send success response with updated balance
	response := map[string]interface{}{
		"message":    "Withdrawal successful",
		"amount":     withdrawReqBody.AmountToWithdraw,
		"newBalance": player.Wallet - withdrawReqBody.AmountToWithdraw,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
