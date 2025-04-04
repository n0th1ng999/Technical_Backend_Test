package models

import (
	"database/sql"
	"fmt"
	"main/events"
	"math"

	_ "modernc.org/sqlite"
)

type Player struct {
	ID         int     `json:"id"`
	Password   string  `json:"password"`
	Name       string  `json:"name"`
	Wallet     float32 `json:"wallet"`     // FLOAT 32 to avoid crazy floating point issues
	BetBalance float32 `json:"betBalance"` // + I don't think anybody has more than 2,147,483,647 in their account xD
	IsBetting  bool    `json:"isBetting"`
}

// Deducts the bet amount, prioritizing bet balance over wallet
func (p *Player) DeductBetAmount(betAmount float32) error {

	// Deduct from bet balance first
	if betAmount <= p.BetBalance {
		p.BetBalance -= betAmount
	} else {
		// Deduct what's available from bet balance and then the rest
		remaining := betAmount - p.BetBalance
		p.BetBalance = 0
		p.Wallet -= remaining
	}

	return nil
}

// RegisterPlayer stores player data and returns the player ID
func RegisterPlayer(playerName string, hashedPlayerPassword string) (int, error) {
	// Query to insert new player and return the auto-generated ID
	query := `INSERT INTO players (name, password) 
	          VALUES (?, ?);`

	// Execute the query and get the last inserted ID
	result, err := DB.Exec(query, playerName, hashedPlayerPassword)
	if err != nil {
		return 0, err
	}

	// Retrieve the last inserted ID
	playerID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("unable to retrieve last inserted ID: %v", err)
	}

	return int(playerID), nil
}
func GetPlayerByID(id int) (*Player, error) {
	query := `SELECT id, name, password, wallet, betBalance, isBetting FROM players WHERE id = ?;`
	var player Player
	err := DB.QueryRow(query, id).Scan(&player.ID, &player.Name, &player.Password, &player.Wallet, &player.BetBalance, &player.IsBetting)
	if err != nil {
		if err == sql.ErrNoRows {
			// Player not found
			return nil, fmt.Errorf("player with ID %d not found", id)
		}
		return nil, fmt.Errorf("error fetching player: %v", err)
	}
	return &player, nil
}

func GetPlayerByName(name string) (*Player, error) {

	query := "SELECT id, password FROM players WHERE name = ?"
	var player Player
	err := DB.QueryRow(query, name).Scan(&player.ID, &player.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			// Player not found with the given name
			return nil, fmt.Errorf("player with name '%s' not found", name)
		}
		return nil, fmt.Errorf("error fetching player: %v", err)
	}

	return &player, nil
}

func GetPlayerByUsernameAndPassword(username, password string) (*Player, error) {
	query := `SELECT id, name, password, wallet, betBalance, isBetting 
	          FROM players WHERE name = ? AND password = ?;`

	var player Player
	err := DB.QueryRow(query, username, password).Scan(&player.ID, &player.Name, &player.Password, &player.Wallet, &player.BetBalance, &player.IsBetting)
	if err != nil {
		if err == sql.ErrNoRows {
			// Player not found with the given username and password
			return nil, fmt.Errorf("player with username '%s' and the provided password not found", username)
		}
		return nil, fmt.Errorf("error fetching player: %v", err)
	}

	return &player, nil
}

func CheckPlayerBettingStatus(id int) (bool, error) {
	var isBetting bool
	query := `SELECT isBetting FROM players WHERE id = ?;`
	err := DB.QueryRow(query, id).Scan(&isBetting)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("player with ID %d not found", id)
		}
		return false, fmt.Errorf("error checking player playing status: %v", err)
	}
	return isBetting, nil
}

func UpdatePlayerBalance(playerId int, newWalletBalance float32, newBetBalance float32) error {

	// Format values to have 2 decimal places
	newWalletBalance = float32(math.Round(float64(newWalletBalance)*100) / 100)
	newBetBalance = float32(math.Round(float64(newBetBalance)*100) / 100)
	
	// Update the player's wallet and bet balance
	// This will also emit an event to notify listeners about the balance update

	query := `UPDATE players SET wallet =?, betBalance =? WHERE id =?;`
	_, err := DB.Exec(query, newWalletBalance, newBetBalance, playerId)
	if err != nil {

		return err
	}

	// Emit Event for balance update
	balanceUpdateEvent := fmt.Sprintf("BalanceUpdate_%d", playerId)

	// Prepare the balance data to send with the event
	var betData = events.EventWalletData{
		Wallet:     newWalletBalance,
		BetBalance: newBetBalance,
	}

	// Emit the event so listeners can react to it (e.g., update WebSocket clients)
	events.GlobalEmitter.Emit(balanceUpdateEvent, betData)

	return nil
}

func UpdatePlayerBettingStatus(id int, isBetting bool) error {
	query := `UPDATE players SET isBetting =? WHERE id =?;`

	_, err := DB.Exec(query, isBetting, id)
	if err != nil {
		return fmt.Errorf("player betting status was not updated")
	}

	return nil
}
