package main

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Get database URL from environment variable
	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		log.Fatal("POSTGRES_URL environment variable is required")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		os.Exit(1)
	}

	log.Println("Starting cleanup of duplicate clan members...")

	// Find players who are in multiple clans
	var duplicates []struct {
		PlayerTag string
		Count     int64
	}

	if err := db.Raw(`
		SELECT player_tag, COUNT(*) as count 
		FROM clan_members 
		GROUP BY player_tag 
		HAVING COUNT(*) > 1
	`).Scan(&duplicates).Error; err != nil {
		log.Printf("Failed to find duplicate members: %v", err)
		os.Exit(1)
	}

	log.Printf("Found %d players in multiple clans", len(duplicates))

	for _, duplicate := range duplicates {
		log.Printf("Cleaning up player %s (in %d clans)", duplicate.PlayerTag, duplicate.Count)

		// Keep only the most recently added membership
		if err := db.Exec(`
			DELETE FROM clan_members 
			WHERE player_tag = ? 
			AND (player_tag, clan_tag) NOT IN (
				SELECT player_tag, clan_tag 
				FROM clan_members 
				WHERE player_tag = ? 
				ORDER BY clan_tag 
				LIMIT 1
			)
		`, duplicate.PlayerTag, duplicate.PlayerTag).Error; err != nil {
			log.Printf("Failed to clean up player %s: %v", duplicate.PlayerTag, err)
		} else {
			log.Printf("Successfully cleaned up player %s", duplicate.PlayerTag)
		}
	}

	log.Println("Cleanup completed!")
}
