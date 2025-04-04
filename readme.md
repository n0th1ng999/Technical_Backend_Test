# Technical Backend Test for Vertsa Play

## Overview
This project is a backend system written in **Go** that utilizes **WebSockets** to facilitate a dice-based betting game.

## Prerequisites
Ensure you have Go installed on your system. You can download it from [Go's official website](https://go.dev/dl/).

## Running the Project
### Install Dependencies
```sh
cd ./Backend

go mod tidy
```

### Start the Backend
```sh
cd ./Backend

go run main.go
```

### Start with Hot Reload
For development convenience, use [Air] (https://github.com/air-verse/air) or Similar :
```sh
cd ./Backend

air
```

## Configuration
Modify the `config.env` file to adjust server settings:

```env
PORT=:8080  # Port for the server
RIGGED_DICE_NUMBER=  # Predetermined dice roll result (optional)
WINNING_MULTIPLIER=2  # Multiplier for bet winnings
SOCKET_TIMEOUT_DURATION=3600  # Timeout for WebSocket connections (in seconds)
PROCESSING_DURATION=2  # Processing time for game actions
JWT_SECRET=A_SECRET  # Secret key for JWT authentication
JWT_DURATION_IN_HOURS=24  # Expiration time for JWT tokens (in hours)
```

## Feature List
### Wallet Management
- [x] Wallet endpoint
  - Retrieves wallet balance on first request
  - Updates balance after each "play" request
  - Updates balance after an "end play" request
  - Updates balance after a "Wallet Withdraw" request
  - Updates balance after a "Wallet Deposit" request

### Game Mechanics
- [x] **Play - Bet on the dice game**
  - Only allows bets up to the wallet's available balance
  - Minimum bet requirement of greater than 0
  - Prevents multiple simultaneous plays or cash-ins, even on parallel sockets

- [x] **End Play - Transfer winnings to wallet**
  - Transfers winnings after play completion if they are >= 0 and do not exceed players bet balance
  - Blocks new bets or cash-ins during processing, even on parallel sockets

### Architecture
- [x] **MVC Structuring**
- [x] **SQLite Database Integration**
  - Initializes Player Table
  - Adds mock data if not already present

### Documentation & Testing
- [x] API documentation and testing with **Postman** (see Postman documentation for details)

## Security Measures
- [x] Authentication via **JWT Auth** (required for all protected endpoints)
  - Disconnects unauthorized users from WebSockets
  - Secures Wallet, Play, and EndPlay WS endpoints, player/me/wallet/deposit and player/me/wallet/withdraw
- [x] Rejects malformed messages (accepts only valid JSON)
- [x] Implements global timeout for each WebSocket connection

## Additional Features
- [x] **Wallet Balance Endpoint**
  - Enables **withdrawal** from the wallet
  - Allows **deposit** to the wallet

### Configurability
- [x] Adjustable game settings:
  - Rigging odds (force a specific dice outcome)
  - Winning multipliers
  - Connection timeouts
  - Server port configuration

## Future Enhancements
### Frontend Development
- [ ] Implement betting UI
- [ ] Collect winnings via frontend
- [ ] Display wallet balance 


---
This README serves as an overview of the backend test project for Vertsa Play. Contributions and improvements are welcome!