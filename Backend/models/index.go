package models

import (
	"database/sql"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// DB is a global variable to hold the database connection
var DB *sql.DB

// ConnectDB initializes the database connection
func ConnectDB() {
	var err error

	// Open a connection to SQLite
	DB, err = sql.Open("sqlite", "database.db") // Change "database.db" to your file name
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// SQLite doesn't support concurrency well -> Limit to 1 Open Conn
	// No transactions are needed for this project
	// because there can't be simultaneous queries being executed
	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)

	// Check if the connection is working
	if err = DB.Ping(); err != nil {
		log.Fatal("Database is not accessible:", err)
	}
	log.Println("Connected to SQLite database successfully")

	initializeTables()
}

func CloseDB() {
	DB.Close()
}

func initializeTables() {
	// Use Exec instead of Query for table creation
	query := `
	CREATE TABLE IF NOT EXISTS players (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		wallet DECIMAL(10,2) NOT NULL DEFAULT 0,
		betBalance DECIMAL(10,2) DEFAULT 0,
        isBetting BOOLEAN DEFAULT false
	);`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Error creating table:", err)
	}

	fmt.Println("TABLE Players Initialized Successfully")

	// Check if the table has any players
	var count int
	err = DB.QueryRow("SELECT COUNT(*) FROM players;").Scan(&count)
	if err != nil {
		log.Fatal("Error checking player count:", err)
	}

	// Insert mock data only if no players exist
	if count == 0 {
		insertMockData()
	}
}

// insertMockData adds test players to the database
func insertMockData() {
	// This is only for testing purposes
	// Real Passwords
	passwords := []string{"password123", "securepass", "mypassword"}
	// Hashed Passwords
	hashedPasswords := []any{}

	for _, password := range passwords {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Error hashing password:", err)
		}
		hashedPasswords = append(hashedPasswords, string(hashedPassword))
	}

	query := `INSERT INTO players (name, password, wallet, betBalance, isBetting) VALUES 
		('Alice', ?, 500.00, 0.00, false), 
		('Bob', ?, 300.00, 30.00, false),
		('Charlie', ?, 700.00, 70.00, true);`

	_, err := DB.Exec(query, hashedPasswords...)
	if err != nil {
		log.Fatal("Error inserting mock data:", err)
	}

	fmt.Println("Mock data inserted successfully")
}
