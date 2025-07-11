package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Player struct {
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	Provider        string    `json:"provider"`
	ProviderID      string    `json:"provider_id"`
	Level           int       `json:"level"`
	Experience      int       `json:"experience"`
	Coins           int       `json:"coins"`
	TotalFishCaught int       `json:"total_fish_caught"`
	LastAutoFish    *time.Time `json:"last_auto_fish"`
	NextAutoFish    *time.Time `json:"next_auto_fish"`
	CreatedAt       time.Time `json:"created_at"`
}

type Fish struct {
	Name     string    `json:"fish_name"`
	Rarity   string    `json:"rarity"`
	Value    int       `json:"value"`
	CaughtAt time.Time `json:"caught_at"`
}

type FishType struct {
	Name   string `json:"name"`
	Rarity string `json:"rarity"`
	Value  int    `json:"value"`
}

type PlayerRegister struct {
	Email      string `json:"email" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Provider   string `json:"provider" binding:"required"`
	ProviderID string `json:"provider_id" binding:"required"`
}

type PlayerLogin struct {
	Email      string `json:"email" binding:"required"`
	Provider   string `json:"provider" binding:"required"`
	ProviderID string `json:"provider_id" binding:"required"`
}

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

var (
	playersDB      = make(map[string]*Player)
	fishInventory  = make(map[string][]Fish)
	playersMutex   = sync.RWMutex{}
	inventoryMutex = sync.RWMutex{}
	jwtSecret      = []byte("your-secret-key-change-in-production")
	
	fishTypes = []FishType{
		{"Common Carp", "common", 10},
		{"Bass", "common", 15},
		{"Trout", "uncommon", 25},
		{"Salmon", "uncommon", 30},
		{"Tuna", "rare", 50},
		{"Swordfish", "rare", 75},
		{"Golden Fish", "legendary", 200},
		{"Dragon Fish", "legendary", 500},
	}
	
	rarityWeights = map[string]int{
		"common":    50,
		"uncommon":  30,
		"rare":      15,
		"legendary": 5,
	}
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	go autoFishingScheduler()

	r.GET("/healthz", healthCheck)
	r.POST("/auth/register", registerPlayer)
	r.POST("/auth/login", loginPlayer)
	r.GET("/player/profile", authMiddleware(), getPlayerProfile)
	r.GET("/player/inventory", authMiddleware(), getPlayerInventory)
	r.POST("/fishing/manual", authMiddleware(), manualFishing)
	r.GET("/fishing/status", authMiddleware(), getFishingStatus)
	r.GET("/game/fish-types", getFishTypes)
	r.GET("/game/leaderboard", getLeaderboard)

	fmt.Println("🎣 Auto Fishing Game Server starting on :8080")
	log.Fatal(r.Run(":8080"))
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func createJWTToken(email string) (string, error) {
	expirationTime := time.Now().Add(7 * 24 * time.Hour)
	claims := &Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("email", claims.Email)
		c.Next()
	}
}

func catchRandomFish() Fish {
	totalWeight := 0
	for _, weight := range rarityWeights {
		totalWeight += weight
	}

	randomNum := rand.Intn(totalWeight) + 1
	currentWeight := 0
	selectedRarity := "common"

	for rarity, weight := range rarityWeights {
		currentWeight += weight
		if randomNum <= currentWeight {
			selectedRarity = rarity
			break
		}
	}

	var fishOfRarity []FishType
	for _, fish := range fishTypes {
		if fish.Rarity == selectedRarity {
			fishOfRarity = append(fishOfRarity, fish)
		}
	}

	selectedFish := fishOfRarity[rand.Intn(len(fishOfRarity))]

	return Fish{
		Name:     selectedFish.Name,
		Rarity:   selectedFish.Rarity,
		Value:    selectedFish.Value,
		CaughtAt: time.Now(),
	}
}

func performAutoFishing(playerEmail string) {
	playersMutex.Lock()
	player, exists := playersDB[playerEmail]
	if !exists {
		playersMutex.Unlock()
		return
	}

	fish := catchRandomFish()

	inventoryMutex.Lock()
	if fishInventory[playerEmail] == nil {
		fishInventory[playerEmail] = []Fish{}
	}
	fishInventory[playerEmail] = append(fishInventory[playerEmail], fish)
	inventoryMutex.Unlock()

	player.TotalFishCaught++
	player.Experience += fish.Value
	player.Coins += fish.Value
	now := time.Now()
	player.LastAutoFish = &now
	nextFish := now.Add(30 * time.Minute)
	player.NextAutoFish = &nextFish

	newLevel := player.Experience/100 + 1
	if newLevel > player.Level {
		player.Level = newLevel
	}

	playersMutex.Unlock()

	fmt.Printf("🎣 Auto-fishing for %s: caught %s (%s)\n", playerEmail, fish.Name, fish.Rarity)
}

func autoFishingScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentTime := time.Now()
			
			playersMutex.RLock()
			var playersToFish []string
			for email, player := range playersDB {
				if player.NextAutoFish != nil && currentTime.After(*player.NextAutoFish) {
					playersToFish = append(playersToFish, email)
				}
			}
			playersMutex.RUnlock()

			for _, email := range playersToFish {
				go performAutoFishing(email)
			}
		}
	}
}

func registerPlayer(c *gin.Context) {
	var req PlayerRegister
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playersMutex.Lock()
	defer playersMutex.Unlock()

	if _, exists := playersDB[req.Email]; exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Player already exists"})
		return
	}

	nextAutoFish := time.Now().Add(30 * time.Minute)
	player := &Player{
		Email:           req.Email,
		Name:            req.Name,
		Provider:        req.Provider,
		ProviderID:      req.ProviderID,
		Level:           1,
		Experience:      0,
		Coins:           100,
		TotalFishCaught: 0,
		LastAutoFish:    nil,
		NextAutoFish:    &nextAutoFish,
		CreatedAt:       time.Now(),
	}

	playersDB[req.Email] = player

	inventoryMutex.Lock()
	fishInventory[req.Email] = []Fish{}
	inventoryMutex.Unlock()

	token, err := createJWTToken(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Player registered successfully",
		"token":   token,
		"player":  player,
	})
}

func loginPlayer(c *gin.Context) {
	var req PlayerLogin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playersMutex.RLock()
	player, exists := playersDB[req.Email]
	playersMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		return
	}

	if player.Provider != req.Provider {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider"})
		return
	}

	token, err := createJWTToken(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"player":  player,
	})
}

func getPlayerProfile(c *gin.Context) {
	email := c.GetString("email")
	
	playersMutex.RLock()
	player, exists := playersDB[email]
	playersMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		return
	}

	c.JSON(http.StatusOK, player)
}

func getPlayerInventory(c *gin.Context) {
	email := c.GetString("email")
	
	inventoryMutex.RLock()
	inventory, exists := fishInventory[email]
	if !exists {
		inventory = []Fish{}
	}
	inventoryMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"inventory":  inventory,
		"total_fish": len(inventory),
	})
}

func manualFishing(c *gin.Context) {
	email := c.GetString("email")
	
	playersMutex.Lock()
	player, exists := playersDB[email]
	if !exists {
		playersMutex.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		return
	}

	if player.Coins < 10 {
		playersMutex.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough coins for manual fishing"})
		return
	}

	player.Coins -= 10
	fish := catchRandomFish()

	inventoryMutex.Lock()
	if fishInventory[email] == nil {
		fishInventory[email] = []Fish{}
	}
	fishInventory[email] = append(fishInventory[email], fish)
	inventoryMutex.Unlock()

	player.TotalFishCaught++
	player.Experience += fish.Value
	player.Coins += fish.Value

	newLevel := player.Experience/100 + 1
	if newLevel > player.Level {
		player.Level = newLevel
	}

	playersMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message": "Fish caught!",
		"fish":    fish,
		"player_stats": gin.H{
			"level":             player.Level,
			"experience":        player.Experience,
			"coins":             player.Coins,
			"total_fish_caught": player.TotalFishCaught,
		},
	})
}

func getFishingStatus(c *gin.Context) {
	email := c.GetString("email")
	
	playersMutex.RLock()
	player, exists := playersDB[email]
	playersMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		return
	}

	var timeUntilNext string
	if player.NextAutoFish != nil {
		if time.Now().After(*player.NextAutoFish) {
			timeUntilNext = "Ready now!"
		} else {
			timeDiff := player.NextAutoFish.Sub(time.Now())
			minutesLeft := int(timeDiff.Minutes())
			timeUntilNext = strconv.Itoa(minutesLeft) + " minutes"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"last_auto_fish":       player.LastAutoFish,
		"next_auto_fish":       player.NextAutoFish,
		"time_until_next":      timeUntilNext,
		"auto_fishing_enabled": true,
	})
}

func getFishTypes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"fish_types":     fishTypes,
		"rarity_weights": rarityWeights,
	})
}

func getLeaderboard(c *gin.Context) {
	playersMutex.RLock()
	defer playersMutex.RUnlock()

	type LeaderboardEntry struct {
		Rank            int    `json:"rank"`
		Name            string `json:"name"`
		Level           int    `json:"level"`
		TotalFishCaught int    `json:"total_fish_caught"`
		Experience      int    `json:"experience"`
	}

	var entries []LeaderboardEntry
	for _, player := range playersDB {
		entries = append(entries, LeaderboardEntry{
			Name:            player.Name,
			Level:           player.Level,
			TotalFishCaught: player.TotalFishCaught,
			Experience:      player.Experience,
		})
	}

	for i := 0; i < len(entries)-1; i++ {
		for j := 0; j < len(entries)-i-1; j++ {
			if entries[j].TotalFishCaught < entries[j+1].TotalFishCaught {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}

	for i := range entries {
		entries[i].Rank = i + 1
		if i >= 10 {
			break
		}
	}

	if len(entries) > 10 {
		entries = entries[:10]
	}

	c.JSON(http.StatusOK, gin.H{"leaderboard": entries})
}
