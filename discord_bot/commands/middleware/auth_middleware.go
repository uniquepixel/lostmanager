package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"

	"bot/commands/messages"
	"bot/commands/repos"
	"bot/store/postgres/models"
	"bot/types"
)

type AuthMiddleware struct {
	guilds repos.IGuildsRepo
	clans  repos.IClansRepo
	users  repos.IUsersRepo
}

func NewAuthMiddleware(guilds repos.IGuildsRepo, clans repos.IClansRepo, users repos.IUsersRepo) AuthMiddleware {
	return AuthMiddleware{
		guilds: guilds,
		clans:  clans,
		users:  users,
	}
}

// AuthorizeAdminInteraction checks if the user is an admin and sends an error message if not. Returns an error if the user is not an admin
func (m *AuthMiddleware) AuthorizeAdminInteraction(i *discordgo.InteractionCreate) error {
	user, err := m.users.UserByDiscordID(i.Member.User.ID)
	if err != nil {
		slog.Error("Error getting user by discord ID.", slog.Any("err", err))
		messages.SendUnknownErr(i)
		return err
	}
	if user.IsAdmin {
		return nil
	}
	m.sendAdminRequired(i)
	return errors.New("member is not an admin")
}

// AuthorizeInteraction checks if the user has the required role in the clan and sends an error message if not. Returns an error if the user doesn't have permission.
func (m *AuthMiddleware) AuthorizeInteraction(i *discordgo.InteractionCreate, clanTag string, role types.AuthRole) error {
	user := models.UserFromGuildMember(i.Member)
	if err := m.users.CreateOrUpdateUser(user); err != nil {
		messages.SendUnknownErr(i)
		return err
	}
	if user.IsAdmin {
		return nil
	}
	if role == types.AuthRoleAdmin {
		m.sendAdminRequired(i)
		return errors.New("member is not an admin")
	}

	guild, err := m.guilds.GuildByClanTag(i.GuildID, clanTag)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			m.sendClanNotInGuildError(i, clanTag)
			return err
		}

		slog.Error("Error getting guild by clan tag.", slog.Any("err", err))
		messages.SendUnknownErr(i)
		return err
	}

	switch role {
	case types.AuthRoleMember:
		if guild.IsMember(i.Member.Roles) {
			return nil
		}
	case types.AuthRoleElder:
		if guild.IsElder(i.Member.Roles) {
			return nil
		}
	case types.AuthRoleCoLeader:
		if guild.IsCoLeader(i.Member.Roles) {
			return nil
		}
	case types.AuthRoleLeader:
		if guild.IsLeader(i.Member.Roles) {
			return nil
		}
	}

	m.sendNoPermissionError(i, clanTag, role)
	return errors.New("member is not a leader")
}

func (m *AuthMiddleware) AuthorizeInteractionWithoutMessageEditNoClanNeeded(s *discordgo.Session, i *discordgo.InteractionCreate, clanTag string, role types.AuthRole) error {
	user := models.UserFromGuildMember(i.Member)

	if err := m.users.CreateOrUpdateUser(user); err != nil {
		messages.CreateAndEditEmbed(s, i, "Unbekannter Fehler", "Ein unbekannter Fehler ist aufgetreten.", messages.ColorRed)
		return err
	}
	if user.IsAdmin {
		return nil
	}
	if role == types.AuthRoleAdmin {
		messages.CreateAndEditEmbed(s, i, "Keine Berechtigung", "Um diesen Befehl ausführen zu können, musst du ein Administrator sein.", messages.ColorRed)
		return errors.New("member is not an admin")
	}

	guild, err := m.guilds.GuildByClanTag(i.GuildID, clanTag)
	if err != nil {
		guilds, _ := m.guilds.Guilds(models.LostFamilyGuildID)

		var LeaderRoleIDs []string
		var CoLeaderRoleIDs []string
		var ElderRoleIDs []string
		var MemberRoleIDs []string

		for _, guild := range guilds {
			LeaderRoleIDs = append(LeaderRoleIDs, guild.LeaderRoleID)
			CoLeaderRoleIDs = append(CoLeaderRoleIDs, guild.CoLeaderRoleID)
			ElderRoleIDs = append(ElderRoleIDs, guild.ElderRoleID)
			MemberRoleIDs = append(MemberRoleIDs, guild.MemberRoleID)
		}

		switch role {
		case types.AuthRoleMember:
			for _, r := range i.Member.Roles {
				if slices.Contains(LeaderRoleIDs, r) || slices.Contains(CoLeaderRoleIDs, r) || slices.Contains(ElderRoleIDs, r) || slices.Contains(MemberRoleIDs, r) {
					return nil
				}
			}
		case types.AuthRoleElder:
			for _, r := range i.Member.Roles {
				if slices.Contains(LeaderRoleIDs, r) || slices.Contains(CoLeaderRoleIDs, r) || slices.Contains(ElderRoleIDs, r) {
					return nil
				}
			}
		case types.AuthRoleCoLeader:
			for _, r := range i.Member.Roles {
				if slices.Contains(LeaderRoleIDs, r) || slices.Contains(CoLeaderRoleIDs, r) {
					return nil
				}
			}
		case types.AuthRoleLeader:
			for _, r := range i.Member.Roles {
				if slices.Contains(LeaderRoleIDs, r) {
					return nil
				}
			}
		}
	} else {
		switch role {
		case types.AuthRoleMember:
			if guild.IsMember(i.Member.Roles) {
				return nil
			}
		case types.AuthRoleElder:
			if guild.IsElder(i.Member.Roles) {
				return nil
			}
		case types.AuthRoleCoLeader:
			if guild.IsCoLeader(i.Member.Roles) {
				return nil
			}
		case types.AuthRoleLeader:
			if guild.IsLeader(i.Member.Roles) {
				return nil
			}
		}
	}

	// clan, err := m.clans.ClanByTag(clanTag)
	// if err != nil {
	// 	if errors.Is(err, gorm.ErrRecordNotFound) {
	// 		messages.CreateAndEditEmbed(s, i, "Ungültiger Clan", fmt.Sprintf("Der Clan mit dem ClanTag '%s' ist nicht Teil dieses Discord Servers.", clanTag), messages.ColorRed)
	// 	} else {
	// 		slog.Error("Error getting clan by tag.", slog.Any("err", err))
	// 		messages.CreateAndEditEmbed(s, i, "Unbekannter Fehler", "Ein unbekannter Fehler ist aufgetreten.", messages.ColorRed)
	// 	}
	// 	return err
	// }

	// messages.CreateAndEditEmbed(s, i, "Keine Berechtigung", fmt.Sprintf("Um diesen Befehl ausführen zu können, musst du %s sein.", role.String()), messages.ColorRed)
	return errors.New("member is not a leader")
}

func (m *AuthMiddleware) sendClanNotInGuildError(i *discordgo.InteractionCreate, clanTag string) {
	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Ungültiger Clan",
		fmt.Sprintf("Der Clan mit dem ClanTag '%s' ist nicht Teil dieses Discord Servers.", clanTag),
		messages.ColorRed,
	))
}

func (m *AuthMiddleware) sendAdminRequired(i *discordgo.InteractionCreate) {
	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Keine Berechtigung",
		"Um diesen Befehl ausführen zu können, musst du ein Administrator sein.",
		messages.ColorRed,
	))
}

func (m *AuthMiddleware) sendNoPermissionError(i *discordgo.InteractionCreate, clanTag string, role types.AuthRole) {
	clan, err := m.clans.ClanByTag(clanTag)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendClanNotFound(i, clanTag)
			return
		}
		messages.SendUnknownErr(i)
		return
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Keine Berechtigung",
		fmt.Sprintf("Um diesen Befehl ausführen zu können, musst du %s in %s sein.", role.String(), clan.Name),
		messages.ColorRed,
	))
}
