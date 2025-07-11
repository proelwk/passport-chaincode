# Golang Auto-Fishing Game Backend

A complete auto-fishing game backend implemented in Go with OAuth authentication and automatic 30-minute fishing intervals.

## Features

- **OAuth Authentication**: Support for Google and Apple login
- **Auto-Fishing System**: Automatically catches fish every 30 minutes for each player
- **Fish Types**: 8 different fish with 4 rarity levels (common, uncommon, rare, legendary)
- **Player Management**: Registration, login, profile, and inventory management
- **Manual Fishing**: Players can manually fish for 10 coins
- **Leaderboard**: Top 10 players by total fish caught
- **JWT Authentication**: Secure token-based authentication
- **CORS Support**: Ready for frontend integration

## API Endpoints

### Authentication
- `POST /auth/register` - Register new player
- `POST /auth/login` - Login existing player

### Player Management
- `GET /player/profile` - Get player profile (requires auth)
- `GET /player/inventory` - Get player's fish inventory (requires auth)

### Fishing
- `POST /fishing/manual` - Manual fishing (requires auth, costs 10 coins)
- `GET /fishing/status` - Get auto-fishing status (requires auth)

### Game Data
- `GET /game/fish-types` - Get all fish types and rarity weights
- `GET /game/leaderboard` - Get top 10 players leaderboard
- `GET /healthz` - Health check

## Running the Server

```bash
go mod tidy
go run main.go
```

Server will start on port 8080.

## Testing

```bash
# Health check
curl -X GET "http://localhost:8080/healthz"

# Register player
curl -X POST "http://localhost:8080/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "name": "Test Player",
    "provider": "google",
    "provider_id": "google123"
  }'

# Get fish types
curl -X GET "http://localhost:8080/game/fish-types"
```

## Game Mechanics

- Players start with 100 coins and level 1
- Auto-fishing occurs every 30 minutes automatically
- Manual fishing costs 10 coins but gives immediate results
- Fish values range from 10 (Common Carp) to 500 (Dragon Fish)
- Players gain experience and coins equal to fish value
- Level up occurs every 100 experience points
- Rarity distribution: Common (50%), Uncommon (30%), Rare (15%), Legendary (5%)

## Architecture

- **Framework**: Gin (Go web framework)
- **Authentication**: JWT tokens with 7-day expiration
- **Database**: In-memory storage for proof of concept
- **Concurrency**: Thread-safe with mutex locks
- **Scheduler**: Background goroutine for auto-fishing

## Link to Devin run
https://app.devin.ai/sessions/2cf520c5643e473884b57d1bffd3aa66

## Requested by
@proelwk
