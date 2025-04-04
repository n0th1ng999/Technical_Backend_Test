package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Global variables accessible across packages
var (
	PORT                    string
	RIGGED_DICE_NUMBER      int
	WINNING_MULTIPLIER      float32
	SOCKET_TIMEOUT_DURATION float32
	PROCESSING_DURATION     float32
	JWT_SECRET              string
	JWT_DURATION_IN_HOURS   float32
)

// LoadConfig reads environment variables from .env file
func LoadConfig() {
	err := godotenv.Load() // Load .env file
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Read and parse environment variables
	PORT = os.Getenv("PORT") // PORT is a string, no conversion needed

	JWT_SECRET = os.Getenv("JWT_SECRET") // PORT is a string, no conversion needed

	// Convert RIGGED_DICE_NUMBER to an float64 (defaults to 0 if empty or invalid)
	if value, err := strconv.Atoi(os.Getenv("RIGGED_DICE_NUMBER")); err == nil {
		RIGGED_DICE_NUMBER = value
	} else {
		RIGGED_DICE_NUMBER = 0 // Default value if not provided
	}

	// Convert WINNING_MULTIPLIERs to a float64 (defaults to 1.0 if empty or invalid)
	if value, err := strconv.ParseFloat(os.Getenv("WINNING_MULTIPLIER"), 32); err == nil {
		WINNING_MULTIPLIER = float32(value)
	} else {
		WINNING_MULTIPLIER = 1.0 // Default multiplier
	}

	// Convert SOCKET_TIMEOUT_DURATION to an integer (defaults to 10 if empty or invalid)
	if value, err := strconv.ParseFloat(os.Getenv("SOCKET_TIMEOUT_DURATION"), 32); err == nil {
		SOCKET_TIMEOUT_DURATION = float32(value)
	} else {
		SOCKET_TIMEOUT_DURATION = 10 // Default timeout
	}

	if value, err := strconv.ParseFloat(os.Getenv("PROCESSING_DURATION"), 32); err == nil {
		PROCESSING_DURATION = float32(value)
	} else {
		PROCESSING_DURATION = 1 // Default timeout
	}

	if value, err := strconv.ParseFloat(os.Getenv("JWT_DURATION_IN_HOURS"), 32); err == nil {
		JWT_DURATION_IN_HOURS = float32(value)
	} else {
		JWT_DURATION_IN_HOURS = 1 // Default timeout
	}

	fmt.Println("CONFIG LOADED:")
	fmt.Println("	PORT:", PORT)
	fmt.Println("	RIGGED DICE NUMBER:", RIGGED_DICE_NUMBER)
	fmt.Println("	WINNING MULTIPLIER:", WINNING_MULTIPLIER)
	fmt.Println("	SOCKET TIMEOUT DURATION:", SOCKET_TIMEOUT_DURATION)
	fmt.Println("	PROCESSING DURATION:", PROCESSING_DURATION)
	fmt.Println("	JWT SECRET:", JWT_SECRET)
	fmt.Println("	JWT DURATION IN HOURS:", JWT_DURATION_IN_HOURS)
	fmt.Println("\n\n")
}
