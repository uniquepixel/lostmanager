package models

import (
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type User struct {
	DiscordID string `gorm:"size:19;primaryKey;not null"`
	Name      string `gorm:"size:32;not null"`
	IsAdmin   bool   `gorm:"default:false;not null"`
}

func (*User) BeforeCreate(tx *gorm.DB) (err error) {
	tx.Omit("is_admin")
	return nil
}

func (*User) BeforeUpdate(tx *gorm.DB) (err error) {
	tx.Omit("is_admin")
	return nil
}

func (u *User) Mention() string {
	return "<@" + u.DiscordID + ">"
}

func UserFromGuildMember(member *discordgo.Member) *User {
	return &User{
		DiscordID: member.User.ID,
		Name:      member.Nick,
	}
}
