package models

import "time"

type Kickpoint struct {
	ID                 uint      `gorm:"primaryKey;autoIncrement;not null"`
	PlayerTag          string    `gorm:"size:12;not null"`
	ClanTag            string    `gorm:"size:12;not null"`
	Date               time.Time `gorm:"not null"`
	Amount             int       `gorm:"not null"`
	Description        string    `gorm:"size:100"`
	CreatedByDiscordID string    `gorm:"size:18;not null"`
	UpdatedByDiscordID string    `gorm:"size:18"`

	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time

	// Member        *ClanMember `gorm:"foreignKey:PlayerTag,ClanTag;references:PlayerTag,ClanTag"`
	Clan          *Clan   `gorm:"foreignKey:Tag;references:ClanTag"`
	Player        *Player `gorm:"foreignKey:CocTag;references:PlayerTag"`
	CreatedByUser *User   `gorm:"foreignKey:DiscordID;references:CreatedByDiscordID"`
	UpdatedByUser *User   `gorm:"foreignKey:DiscordID;references:UpdatedByDiscordID"`
}
