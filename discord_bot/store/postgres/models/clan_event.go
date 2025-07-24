package models

import "time"

type ClanEvent struct {
	ID              uint      `gorm:"primaryKey"`
	ClanTag         string    `gorm:"not null"`
	StatName        string    `gorm:"not null"`
	StartsAt        time.Time `gorm:"not null"`
	EndsAt          time.Time `gorm:"not null"`
	ChannelID       string    `gorm:"not null"`
	WinnerPlayerTag *string   `gorm:"default:null"`

	Clan *Clan `gorm:"foreignKey:ClanTag;references:Tag;constraint:OnDelete:CASCADE"`
}
