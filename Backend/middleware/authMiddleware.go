package middleware

import (
	"errors"
	"main/config"
	"main/models"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// PrintMessage is an exported function that prints a message.
func AuthenticateUser(r *http.Request) (*models.Player, error) {

	// Check for auth header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, errors.New("authorization token missing")
	}

	// Extract the JWT token to remove the "Bearer "
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse and validate the JWT token
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWT_SECRET), nil
	})
	if err != nil {
		return nil, errors.New("invalid JWT token")
	}

	// Check if the claims can be extracted from the JWT Token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {

		return nil, errors.New("invalid token claims")

	}

	// Extract player ID from token claims
	playerIDFloat, ok := claims["id"].(float64)
	if !ok {

		return nil, errors.New("invalid token payload")
	}

	playerID := int(playerIDFloat)

	// Fetch player from database using ID
	player, findPlayerErr := models.GetPlayerByID(playerID)
	if findPlayerErr != nil {

		return nil, errors.New("player not found")
	}

	// Authentication successful
	return player, nil

}
