package models

import (
	"time"
)

type ClanSettings struct {
	ClanTag                   string `gorm:"size:12;primaryKey;not null"`
	MaxKickpoints             int    `gorm:"not null;default:6"`
	MinSeasonWins             int    `gorm:"not null;default:80"`
	KickpointsExpireAfterDays int    `gorm:"not null;default:45"`
	UpdatedAt                 time.Time
	UpdatedByDiscordID        *string

	Clan             *Clan              `gorm:"foreignKey:ClanTag;references:Tag;constraint:OnUpdate:CASCADE;OnDelete:CASCADE"`
	KickpointReasons []*KickpointReason `gorm:"foreignKey:ClanTag;references:ClanTag;constraint:false"`
	UpdatedByUser    *User              `gorm:"foreignKey:UpdatedByDiscordID;references:DiscordID"`
}
