package models

import (
	"github.com/bwmarrin/discordgo"
)

type Clan struct {
	Tag   string `gorm:"primaryKey;not null"`
	Name  string
	Index int

	Settings    *ClanSettings `gorm:"foreignKey:ClanTag;references:Tag;constraint:OnDelete:CASCADE"`
	ClanMembers ClanMembers   `gorm:"foreignKey:ClanTag;references:Tag;constraint:false"`
}

type Clans []Clan

func (clans Clans) Tags() []string {
	tags := make([]string, len(clans))
	for i, clan := range clans {
		tags[i] = clan.Tag
	}
	return tags
}

func (clans Clans) Choices() []*discordgo.ApplicationCommandOptionChoice {
	choices := make([]*discordgo.ApplicationCommandOptionChoice, len(clans))
	for i, clan := range clans {
		choices[i] = &discordgo.ApplicationCommandOptionChoice{
			Name:  clan.Name,
			Value: clan.Tag,
		}
	}
	return choices
}
