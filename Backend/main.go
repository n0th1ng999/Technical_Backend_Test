package main

import (
	"fmt"
	"main/config"
	"main/controllers"
	"main/models"
	"net/http"
)

func main() {
	config.LoadConfig()

	models.ConnectDB()

	http.HandleFunc("/ws/wallet", controllers.HandleWalletWS)
	http.HandleFunc("/ws/play", controllers.HandlePlayWS)
	http.HandleFunc("/ws/end-play", controllers.HandleEndPlayWS)

	// Authentication routes
	http.HandleFunc("/auth/register", controllers.HandleRegister)
	http.HandleFunc("/auth/login", controllers.HandleLogin)

	// Wallet HTTP routes
	http.HandleFunc("/player/me/wallet/withdraw", controllers.HandleWithdraw)
	http.HandleFunc("/player/me/wallet/deposit", controllers.HandleDeposit)

	fmt.Println("\n\nServer started on ", config.PORT)
	if err := http.ListenAndServe(config.PORT, nil); err != nil {
		fmt.Println("\n\nServer failed to start:", err)
	}

	fmt.Println("Server stopped listening")

	// Close database connection when server stops running.
	models.CloseDB()
}
