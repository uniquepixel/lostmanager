package postgres

import (
	"errors"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"bot/env"
	"bot/store/postgres/models"
)

const (
	maxRetries   = 3
	retryTimeout = time.Second * 15
)

func NewClient() (*gorm.DB, error) {
	db, err := newGormClient()
	if err != nil {
		return nil, err
	}

	if err = migrateAndSeedDB(db); err != nil {
		return nil, err
	}

	return db, nil
}

func migrateAndSeedDB(db *gorm.DB) error {
	// First, migrate all tables without foreign key constraints
	if err := db.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(
		// Independent models first
		&models.User{},
		&models.Player{},
		&models.Clan{},
		&models.Guild{},
		
		// Models that depend on the above
		&models.ClanMember{},
		&models.ClanSettings{},
		&models.KickpointReason{},
		
		// Models that depend on ClanMember
		&models.MemberState{},
		&models.Kickpoint{},
		
		// Event-related models
		&models.ClanEvent{},
		&models.ClanEventMember{},
	); err != nil {
		return err
	}

	// Clean up orphaned data
	if err := cleanupOrphanedData(db); err != nil {
		return err
	}

	return nil
}

func cleanupOrphanedData(db *gorm.DB) error {
	// Clean up orphaned data before applying foreign key constraints
	
	// Remove clan_settings records that don't have corresponding clans
	if err := db.Exec(`
		DELETE FROM clan_settings 
		WHERE clan_tag NOT IN (SELECT tag FROM clans)
	`).Error; err != nil {
		log.Printf("Warning: Could not clean orphaned clan_settings: %v", err)
	}
	
	// Clean up duplicate clan members - keep only the most recent one for each player
	// if err := db.Exec(`
	// 	DELETE FROM clan_members 
	// 	WHERE (player_tag, clan_tag) NOT IN (
	// 		SELECT DISTINCT ON (player_tag) player_tag, clan_tag 
	// 		FROM clan_members 
	// 		ORDER BY player_tag, clan_tag
	// 	)
	// `).Error; err != nil {
	// 	log.Printf("Warning: Could not clean up duplicate clan members: %v", err)
	// }
	
	// Remove clan_members records that don't have corresponding clans or players
	if err := db.Exec(`
		DELETE FROM clan_members 
		WHERE clan_tag NOT IN (SELECT tag FROM clans) 
		   OR player_tag NOT IN (SELECT coc_tag FROM players)
	`).Error; err != nil {
		log.Printf("Warning: Could not clean orphaned clan_members: %v", err)
	}
	
	// Remove kickpoints records that don't have corresponding clans or players
	if err := db.Exec(`
		DELETE FROM kickpoints 
		WHERE clan_tag NOT IN (SELECT tag FROM clans) 
		   OR player_tag NOT IN (SELECT coc_tag FROM players)
	`).Error; err != nil {
		log.Printf("Warning: Could not clean orphaned kickpoints: %v", err)
	}
	
	// Remove member_states records that don't have corresponding clan_members
	if err := db.Exec(`
		DELETE FROM member_states 
		WHERE (player_tag, clan_tag) NOT IN (
			SELECT player_tag, clan_tag FROM clan_members
		)
	`).Error; err != nil {
		log.Printf("Warning: Could not clean orphaned member_states: %v", err)
	}
	
	return nil
}

func newGormClient() (client *gorm.DB, err error) {
	dsn := env.POSTGRES_URL.Value()
	loggerMode := logger.Silent
	if env.MODE.Value() != "PROD" {
		loggerMode = logger.Info
	}

	for i := 0; i < maxRetries; i++ {
		if client, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(loggerMode),
			DisableForeignKeyConstraintWhenMigrating: true, // Disable FK constraints during migration
		}); err != nil {
			log.Printf("Failed to connect to database: %v\nRetrying in %s...", err, retryTimeout.String())
			time.Sleep(retryTimeout)
			continue
		}

		// Migrate database schema and clean up data
		if err := migrateAndSeedDB(client); err != nil {
			panic(err)
		}

		log.Println("Connected to postgres database.")
		return client, nil
	}

	return nil, errors.New("max retries reached")
}
