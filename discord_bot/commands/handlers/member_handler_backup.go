package handlers

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/aaantiii/goclash"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"

	"bot/commands/messages"
	"bot/commands/middleware"
	"bot/commands/repos"
	"bot/commands/util"
	"bot/commands/validation"
	"bot/env"
	"bot/store/postgres/models"
	"bot/types"
)

const (
	ClanTagOptionName   = "clan"
	PlayerTagOptionName = "player"
	MemberTagOptionName = "member"
	RoleOptionName      = "role"
)

type IMemberHandler interface {
	ListMembers(s *discordgo.Session, i *discordgo.InteractionCreate)
	ClanMemberStatus(s *discordgo.Session, i *discordgo.InteractionCreate)
	AddMember(s *discordgo.Session, i *discordgo.InteractionCreate)
	EditMember(s *discordgo.Session, i *discordgo.InteractionCreate)
	RemoveMember(s *discordgo.Session, i *discordgo.InteractionCreate)
	TransferMember(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate)
}

type MemberHandler struct {
	members     repos.IMembersRepo
	clans       repos.IClansRepo
	players     repos.IPlayersRepo
	guilds      repos.IGuildsRepo
	auth        middleware.AuthMiddleware
	clashClient *goclash.Client
}
