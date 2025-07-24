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
	Clan          *Clan   `gorm:"foreignKey:ClanTag;references:Tag;constraint:OnDelete:CASCADE"`
	Player        *Player `gorm:"foreignKey:PlayerTag;references:CocTag;constraint:OnDelete:CASCADE"`
	CreatedByUser *User   `gorm:"foreignKey:CreatedByDiscordID;references:DiscordID"`
	UpdatedByUser *User   `gorm:"foreignKey:UpdatedByDiscordID;references:DiscordID"`
}
