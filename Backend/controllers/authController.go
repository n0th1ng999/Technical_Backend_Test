package controllers

import (
	"encoding/json"
	"fmt"
	"main/config"
	"main/helpers"
	"main/models"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type PlayerLogin struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Register a new player
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var response = map[string]interface{}{"message": "Player registered successfully"}

	var newPlayerData PlayerLogin
	err := json.NewDecoder(r.Body).Decode(&newPlayerData)
	if err != nil {
		response["message"] = "Invalid request payload"
		stringifiedResponse, _ := helpers.JsonStringifier(response)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(stringifiedResponse))
		return
	}
	defer r.Body.Close() // Close body after reading

	newPlayerHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPlayerData.Password), bcrypt.DefaultCost)
	if err != nil {
		response["message"] = "Error hashing password"
		stringifiedResponse, _ := helpers.JsonStringifier(response)
		http.Error(w, stringifiedResponse, http.StatusBadRequest)
		return
	}

	newPlayerId, err := models.RegisterPlayer(newPlayerData.Name, string(newPlayerHashedPassword))
	if err != nil {
		response["message"] = fmt.Sprintln("Error registering player: ", err.Error())
		stringifiedResponse, _ := helpers.JsonStringifier(response)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(stringifiedResponse))
		return
	}

	// Generate JWT
	jwtToken, err := generateJWT(newPlayerId)
	if err != nil {
		response["message"] = "Error generating JWT token"
		stringifiedResponse, _ := helpers.JsonStringifier(response)
		http.Error(w, stringifiedResponse, http.StatusBadRequest)
		return
	}

	response["token"] = jwtToken

	// Send response
	stringifiedResponse, _ := helpers.JsonStringifier(response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(stringifiedResponse))

}

// Login a player
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Default response
	var response = map[string]interface{}{"message": "Player logged in successfully"}

	// Login Data
	var loginData PlayerLogin
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		response["message"] = "Invalid request payload"
		stringifiedResponse, _ := helpers.JsonStringifier(response)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(stringifiedResponse))
		return
	}
	defer r.Body.Close() // Close body after reading

	// Retrieve the hashed password from the simulated database
	player, err := models.GetPlayerByName(loginData.Name)
	if err != nil {
		response["message"] = fmt.Sprintln("Player not found: ", err.Error())
		stringifiedResponse, _ := helpers.JsonStringifier(response)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(stringifiedResponse))
		return
	}

	// Compare the provided password with the stored hashed password
	err = bcrypt.CompareHashAndPassword([]byte(player.Password), []byte(loginData.Password))
	if err != nil {

		response["message"] = "Invalid password"
		stringifiedResponse, _ := helpers.JsonStringifier(response)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(stringifiedResponse))
		return
	}

	// Generate JWT token for the player
	jwtToken, err := generateJWT(player.ID)
	if err != nil {
		response["message"] = "Error generating JWT token"
		stringifiedResponse, _ := helpers.JsonStringifier(response)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(stringifiedResponse))
		return
	}

	// Add the JWT token to the response
	response["token"] = jwtToken

	// Send response
	stringifiedResponse, _ := helpers.JsonStringifier(response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(stringifiedResponse))
}

// Generate JWT token
func generateJWT(playerId int) (string, error) {
	claims := jwt.MapClaims{
		"id":  playerId,
		"exp": time.Now().Add(time.Hour * 15).Unix(), // Token expires in config 
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.JWT_SECRET))
}
