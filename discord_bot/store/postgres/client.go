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

	if err = seedDB(db); err != nil {
		return nil, err
	}

	return db, nil
}

func seedDB(db *gorm.DB) error {
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
		}); err != nil {
			log.Printf("Failed to connect to database: %v\nRetrying in %s...", err, retryTimeout.String())
			time.Sleep(retryTimeout)
			continue
		}

		if err := client.AutoMigrate(
			// &models.ClanEventMember{},
			// &models.ClanEvent{},
			// &models.ClanMember{},
			// &models.ClanSettings{},
			// &models.Clan{},
			// &models.Guild{},
			// &models.KickpointReason{},
			&models.Kickpoint{},
			// &models.MemberState{},
			// &models.Player{},
			// &models.User{},
			// &AnotherModel{},
		); err != nil {
			panic(err)
		}

		log.Println("Connected to postgres database.")
		return client, nil
	}

	return nil, errors.New("max retries reached")
}
