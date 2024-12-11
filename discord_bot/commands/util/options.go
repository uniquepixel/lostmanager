package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func StringOptionByName(name string, options []*discordgo.ApplicationCommandInteractionDataOption) string {
	for _, o := range options {
		if o.Name == name {
			return strings.Trim(o.StringValue(), " ")
		}
	}
	return ""
}

func IntOptionByName(name string, options []*discordgo.ApplicationCommandInteractionDataOption) *int {
	for _, o := range options {
		if o.Name == name {
			value := int(o.IntValue())
			return &value
		}
	}
	return nil
}

func UintOptionByName(name string, options []*discordgo.ApplicationCommandInteractionDataOption) *uint {
	for _, o := range options {
		if o.Name == name {
			value := uint(o.IntValue())
			return &value
		}
	}
	return nil
}

func DateTimeOptionByName(name string, options []*discordgo.ApplicationCommandInteractionDataOption) (time.Time, error) {
	for _, o := range options {
		if o.Name == name {
			return ParseDateTimeString(o.StringValue())
		}
	}
	return time.Time{}, nil
}

func RoleOptionByName(name, guildID string, options []*discordgo.ApplicationCommandInteractionDataOption) (*discordgo.Role, error) {
	for _, o := range options {
		if o.Name == name {
			return o.RoleValue(nil, guildID), nil
		}
	}
	return nil, fmt.Errorf("role option %s not found", name)
}

type Emoji struct {
	Name     string
	ID       string
	GlobalID string
}

func EmojiOptionByName(name string, options []*discordgo.ApplicationCommandInteractionDataOption) (*Emoji, error) {
	v := StringOptionByName(name, options)
	if len(v) < 5 {
		return nil, fmt.Errorf("emoji option %s not found", name)
	}

	parsed := strings.Split(v[1:len(v)-1], ":")
	if len(parsed) != 3 {
		return nil, fmt.Errorf("invalid emoji format")
	}

	return &Emoji{
		Name:     parsed[1],
		ID:       parsed[2],
		GlobalID: v[1 : len(v)-1],
	}, nil
}

func ChannelOptionByName(name string, options []*discordgo.ApplicationCommandInteractionDataOption) (*discordgo.Channel, error) {
	for _, o := range options {
		if o.Name == name {
			return o.ChannelValue(nil), nil
		}
	}
	return nil, fmt.Errorf("channel option %s not found", name)
}
