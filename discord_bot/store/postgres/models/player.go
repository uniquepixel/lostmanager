package models

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Player links a Clash of Clans player tag with a Discord ID.
type Player struct {
	CocTag    string `gorm:"not null;primaryKey"`
	Name      string `gorm:"not null"`
	DiscordID string

	Members ClanMembers `gorm:"foreignKey:PlayerTag;references:CocTag"`
}

type Players []*Player

func (players Players) Choices() []*discordgo.ApplicationCommandOptionChoice {
	choices := make([]*discordgo.ApplicationCommandOptionChoice, len(players))
	for i, player := range players {
		name := fmt.Sprintf("%s (%s)", player.Name, player.CocTag)
		
		// Add clan name if player is a member of any clan
		if len(player.Members) > 0 && player.Members[0].Clan != nil {
			name = fmt.Sprintf("%s - %s", name, player.Members[0].Clan.Name)
		}
		
		choices[i] = &discordgo.ApplicationCommandOptionChoice{
			Name:  name,
			Value: player.CocTag,
		}
	}

	return choices
}

func (players Players) Tags() []string {
	tags := make([]string, len(players))
	for i, player := range players {
		tags[i] = player.CocTag
	}

	return tags
}
