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

type IMemberHandler interface {
	ListMembers(s *discordgo.Session, i *discordgo.InteractionCreate)
	ClanMemberStatus(s *discordgo.Session, i *discordgo.InteractionCreate)
	AddMember(s *discordgo.Session, i *discordgo.InteractionCreate)
	RemoveMember(s *discordgo.Session, i *discordgo.InteractionCreate)
	EditMember(s *discordgo.Session, i *discordgo.InteractionCreate)
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

func NewMemberHandler(members repos.IMembersRepo, clans repos.IClansRepo, players repos.IPlayersRepo, guilds repos.IGuildsRepo, auth middleware.AuthMiddleware, clashClient *goclash.Client) IMemberHandler {
	return &MemberHandler{
		members:     members,
		clans:       clans,
		players:     players,
		guilds:      guilds,
		auth:        auth,
		clashClient: clashClient,
	}
}

func (h *MemberHandler) ListMembers(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	if clanTag == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Clan an.")
		return
	}

	clan, err := h.clans.ClanByTagPreload(clanTag)
	if err != nil {
		messages.SendClanNotFound(i, clanTag)
		return
	}

	messages.SendClanMembers(i, clan)
}

func (h *MemberHandler) ClanMemberStatus(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	if clanTag == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Clan an.")
		return
	}

	members, err := h.members.MembersByClanTag(clanTag)
	if err != nil {
		messages.SendUnknownErr(i)
		return
	}

	clan, err := h.clashClient.GetClan(clanTag)
	if err != nil {
		messages.SendCocApiErr(i, err)
		return
	}

	messages.SendClansMembersStatus(i, members, clan)
}

func (h *MemberHandler) AddMember(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	playerTag := util.StringOptionByName(PlayerTagOptionName, opts)
	role := models.ClanRole(util.StringOptionByName(RoleOptionName, opts))

	if clanTag == "" || playerTag == "" || role == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Clan, ein Mitglied und eine Rolle an.")
		return
	}

	if !validation.ValidateClanRole(role) {
		messages.SendInvalidInputErr(i, fmt.Sprintf("Die Rolle %s ist ungültig.", string(role)))
		return
	}

	requiredAuthRole := types.AuthRoleAdmin
	if role == models.RoleMember || role == models.RoleElder {
		requiredAuthRole = types.AuthRoleCoLeader
	} else if role == models.RoleCoLeader {
		requiredAuthRole = types.AuthRoleLeader
	}

	if err := h.auth.AuthorizeInteraction(i, clanTag, requiredAuthRole); err != nil {
		return
	}

	player, err := h.players.PlayerByTag(playerTag)
	if err != nil || player.DiscordID == "" {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Mitglied nicht verifiziert",
			"Das Mitglied muss sich zuerst verifizieren, bevor es zu einem Clan hinzugefügt werden kann.",
			messages.ColorRed,
		))
		return
	}

	if err = h.members.CreateMember(&models.ClanMember{
		PlayerTag:        playerTag,
		ClanTag:          clanTag,
		ClanRole:         role,
		AddedByDiscordID: i.Member.User.ID,
	}); err != nil {
		// Check if the error is because player is already in another clan
		if strings.Contains(err.Error(), "is already a member of clan") {
			messages.SendEmbedResponse(i, messages.NewEmbed(
				"Mitglied bereits in anderem Clan",
				"Dieses Mitglied ist bereits in einem anderen Clan. Ein Spieler kann nur in einem Clan gleichzeitig sein. Bitte entferne das Mitglied zuerst aus seinem aktuellen Clan oder verwende den Transfer-Befehl.",
				messages.ColorRed,
			))
		} else {
			messages.SendEmbedResponse(i, messages.NewEmbed(
				"Es ist ein Fehler aufgetreten",
				"Beim Speichern des Mitglieds ist ein Fehler aufgetreten. Dies kann daran liegen, dass das Mitglied bereits existiert oder ungültige Daten angegeben wurden.",
				messages.ColorRed,
			))
		}
		return
	}

	guild, roleErr := h.guilds.GuildByClanTag(i.GuildID, clanTag)
	if roleErr == nil {
		roleErr = s.GuildMemberRoleAdd(i.GuildID, player.DiscordID, guild.MemberRoleID)
	}

	desc := fmt.Sprintf("Das Mitglied wurde erfolgreich als %s zum Clan hinzugefügt.", role.Format())
	if roleErr != nil {
		desc += "\n\n**ACHTUNG**: Dem Mitglied konnte die Mitglieder-Rolle nicht zugewiesen werden. Bitte weise ihm die Rolle manuell zu."
	}

	if slices.Contains(i.Member.Roles, env.DISCORD_EX_MEMBER_ROLE_ID.Value()) {
		if err = s.GuildMemberRoleRemove(i.GuildID, player.DiscordID, env.DISCORD_EX_MEMBER_ROLE_ID.Value()); err != nil {
			desc += fmt.Sprintf(
				"\n\n**ACHTUNG**: Dem Mitglied konnte %s nicht entfernt werden. Bitte entferne ihm die Rolle manuell.",
				util.MentionRole(env.DISCORD_EX_MEMBER_ROLE_ID.Value()),
			)
		}
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Mitglied hinzugefügt",
		desc,
		messages.ColorGreen,
	))
}

func (h *MemberHandler) RemoveMember(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	memberTag := util.StringOptionByName(MemberTagOptionName, opts)

	if clanTag == "" || memberTag == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Clan und ein Mitglied an.")
		return
	}

	member, err := h.members.MemberByID(memberTag, clanTag)
	if err != nil {
		messages.SendMemberNotFound(i, memberTag, clanTag)
		return
	}

	requiredAuthRole := types.AuthRoleAdmin
	if member.ClanRole == models.RoleMember || member.ClanRole == models.RoleElder {
		requiredAuthRole = types.AuthRoleCoLeader
	} else if member.ClanRole == models.RoleCoLeader {
		requiredAuthRole = types.AuthRoleLeader
	}

	if err = h.auth.AuthorizeInteraction(i, clanTag, requiredAuthRole); err != nil {
		return
	}

	if err = h.members.DeleteMember(member.PlayerTag, member.ClanTag); err != nil {
		messages.SendUnknownErr(i)
		return
	}

	desc := fmt.Sprintf("Das Mitglied %s wurde aus %s entfernt.", member.Player.Name, member.Clan.Name)
	// remove member role
	guild, err := h.guilds.GuildByClanTag(i.GuildID, clanTag)
	if err == nil {
		err = s.GuildMemberRoleRemove(i.GuildID, member.Player.DiscordID, guild.MemberRoleID)
	}
	if err != nil {
		desc += "\n\n**ACHTUNG**: Dem Mitglied konnte die Clan-Rolle nicht entfernt werden. Bitte entferne ihm die Rolle manuell."
	}

	// Since players can only be in one clan, add ex-member role
	if err = s.GuildMemberRoleAdd(i.GuildID, member.Player.DiscordID, env.DISCORD_EX_MEMBER_ROLE_ID.Value()); err != nil {
		desc += fmt.Sprintf(
			"\n\n**ACHTUNG**: Dem Mitglied konnte %s nicht zugewiesen werden. Bitte weise ihm die Rolle manuell zu.",
			util.MentionRole(env.DISCORD_EX_MEMBER_ROLE_ID.Value()),
		)
	}

	if member.ClanRole == models.RoleElder {
		desc += "\n\n**ACHTUNG**: Das Mitglied war Ältester. Bitte entferne ihm die Ältesten-Rolle manuell."
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Mitglied entfernt",
		desc,
		messages.ColorGreen,
	))
}

func (h *MemberHandler) EditMember(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	memberTag := util.StringOptionByName(MemberTagOptionName, opts)
	role := models.ClanRole(util.StringOptionByName(RoleOptionName, opts))

	if clanTag == "" || memberTag == "" || role == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Clan, ein Mitglied und eine Rolle an.")
		return
	}

	if !validation.ValidateClanRole(role) {
		messages.SendInvalidInputErr(i, fmt.Sprintf("Die Rolle %s ist ungültig.", string(role)))
		return
	}

	requiredAuthRole := types.AuthRoleAdmin
	if role == models.RoleMember || role == models.RoleElder {
		requiredAuthRole = types.AuthRoleCoLeader
	} else if role == models.RoleCoLeader {
		requiredAuthRole = types.AuthRoleLeader
	}

	if err := h.auth.AuthorizeInteraction(i, clanTag, requiredAuthRole); err != nil {
		return
	}

	member, err := h.members.MemberByID(memberTag, clanTag)
	if err != nil {
		messages.SendMemberNotFound(i, memberTag, clanTag)
		return
	}

	if member.ClanRole == role {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Mitglied nicht geändert",
			fmt.Sprintf("Das Mitglied %s hat bereits die Rolle %s.", member.Player.Name, role.Format()),
			messages.ColorRed,
		))
		return
	}

	if err = h.members.UpdateMemberRole(member.PlayerTag, member.ClanTag, role); err != nil {
		messages.SendUnknownErr(i)
		return
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Mitglied geändert",
		fmt.Sprintf("Das Mitglied %s hat nun die Rolle %s.", member.Player.Name, role.Format()),
		messages.ColorGreen,
	))
}

func (h *MemberHandler) TransferMember(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	playerTag := util.StringOptionByName(PlayerTagOptionName, opts)
	fromClanTag := util.StringOptionByName("from_clan", opts)
	toClanTag := util.StringOptionByName("to_clan", opts)
	role := models.ClanRole(util.StringOptionByName(RoleOptionName, opts))

	if playerTag == "" || fromClanTag == "" || toClanTag == "" || role == "" {
		messages.SendInvalidInputErr(i, "Bitte gib alle erforderlichen Felder an.")
		return
	}

	if !validation.ValidateClanRole(role) {
		messages.SendInvalidInputErr(i, fmt.Sprintf("Die Rolle %s ist ungültig.", string(role)))
		return
	}

	// Check authorization for both clans (admin required for transfers)
	if err := h.auth.AuthorizeInteraction(i, fromClanTag, types.AuthRoleAdmin); err != nil {
		return
	}
	if err := h.auth.AuthorizeInteraction(i, toClanTag, types.AuthRoleAdmin); err != nil {
		return
	}

	// Verify the player exists in the source clan
	currentMember, err := h.members.MemberByID(playerTag, fromClanTag)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendMemberNotFound(i, playerTag, fromClanTag)
		} else {
			messages.SendUnknownErr(i)
		}
		return
	}

	// Perform the transfer
	if err = h.members.TransferMember(playerTag, fromClanTag, toClanTag, role, i.Member.User.ID); err != nil {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Transfer fehlgeschlagen",
			"Beim Übertragen des Mitglieds ist ein Fehler aufgetreten.",
			messages.ColorRed,
		))
		return
	}

	// Update Discord roles if possible
	player, err := h.players.PlayerByTag(playerTag)
	if err == nil && player.DiscordID != "" {
		// Remove old clan role
		if fromGuild, err := h.guilds.GuildByClanTag(i.GuildID, fromClanTag); err == nil {
			s.GuildMemberRoleRemove(i.GuildID, player.DiscordID, fromGuild.MemberRoleID)
		}

		// Add new clan role
		if toGuild, err := h.guilds.GuildByClanTag(i.GuildID, toClanTag); err == nil {
			s.GuildMemberRoleAdd(i.GuildID, player.DiscordID, toGuild.MemberRoleID)
		}
	}

	fromClanName, _ := h.clans.ClanNameByTag(fromClanTag)
	toClanName, _ := h.clans.ClanNameByTag(toClanTag)

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Mitglied übertragen",
		fmt.Sprintf("Das Mitglied %s wurde erfolgreich von %s zu %s übertragen und hat nun die Rolle %s.",
			currentMember.Player.Name, fromClanName, toClanName, role.Format()),
		messages.ColorGreen,
	))
}

func (h *MemberHandler) HandleAutocomplete(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options

	for _, opt := range opts {
		if !opt.Focused {
			continue
		}

		switch opt.Name {
		case ClanTagOptionName:
			autocompleteClans(i, h.clans, opt.StringValue())
		case MemberTagOptionName:
			autocompleteMembers(i, h.players, opt.StringValue(), util.StringOptionByName(ClanTagOptionName, opts))
		case PlayerTagOptionName:
			autocompletePlayers(i, h.players, opt.StringValue())
		case "from_clan", "to_clan":
			autocompleteClans(i, h.clans, opt.StringValue())
		}
	}
}
