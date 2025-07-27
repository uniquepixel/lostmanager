package commands

import (
	"github.com/aaantiii/goclash"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"

	"bot/commands/handlers"
	"bot/commands/middleware"
	"bot/commands/repos"
	"bot/commands/util"
	"bot/store/postgres/models"
	"bot/types"
)

func memberInteractionCommands(db *gorm.DB, clashClient *goclash.Client) types.Commands[types.InteractionHandler] {
	handler := handlers.NewMemberHandler(
		repos.NewMembersRepo(db),
		repos.NewClansRepo(db),
		repos.NewPlayersRepo(db),
		repos.NewGuildsRepo(db),
		middleware.NewAuthMiddleware(repos.NewGuildsRepo(db), repos.NewClansRepo(db), repos.NewUsersRepo(db)),
		clashClient,
	)

	return types.Commands[types.InteractionHandler]{{
		Handler: types.InteractionHandler{
			Main:         handler.ListMembers,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "listmembers",
			Description:  "Listet alle Mitglieder eines Clans auf.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, dessen Mitglieder aufgelistet werden sollen."),
			},
		},
	}, {
		Handler: types.InteractionHandler{
			Main:         handler.ClanMemberStatus,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "memberstatus",
			Description:  "Vergleicht hinzugefügte Mitglieder der Datenbank mit den ingame-Clan Mitgliedern.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, dessen Mitglieder Status überprüft werden soll."),
			},
		},
	}, {
		Handler: types.InteractionHandler{
			Main:         handler.AddMember,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "addmember",
			Description:  "Fügt ein Mitglied zu einem Clan hinzu.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, zu dem das Mitglied hinzugefügt werden soll."),
				optionPlayerTag("Mitglied, das zum Clan hinzugefügt werden soll."),
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "role",
					Description: "Rolle, die das Mitglied im Clan haben soll.",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: models.RoleLeader.Format(), Value: models.RoleLeader.String()},
						{Name: models.RoleCoLeader.Format(), Value: models.RoleCoLeader.String()},
						{Name: models.RoleElder.Format(), Value: models.RoleElder.String()},
						{Name: models.RoleMember.Format(), Value: models.RoleMember.String()},
					},
				},
			},
		},
	}, {
		Handler: types.InteractionHandler{
			Main:         handler.EditMember,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "editmember",
			Description:  "Ändere die Rolle eines Mitglieds in einem Clan.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, zu dem das Mitglied gehört."),
				optionMemberTag("Mitglied, dessen Rolle bearbeitet werden soll."),
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "role",
					Description: "Rolle, die das Mitglied im Clan haben soll.",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: models.RoleLeader.Format(), Value: models.RoleLeader.String()},
						{Name: models.RoleCoLeader.Format(), Value: models.RoleCoLeader.String()},
						{Name: models.RoleElder.Format(), Value: models.RoleElder.String()},
						{Name: models.RoleMember.Format(), Value: models.RoleMember.String()},
					},
				},
			},
		},
	}, {
		Handler: types.InteractionHandler{
			Main:         handler.RemoveMember,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "removemember",
			Description:  "Mitglied von einem Clan entfernen.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, von dem das Mitglied entfernt werden soll."),
				optionMemberTag("Mitglied, das vom Clan entfernt werden soll."),
			},
		},
	}, {
		Handler: types.InteractionHandler{
			Main:         handler.TransferMember,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "transfermember",
			Description:  "Überträgt ein Mitglied von einem Clan zu einem anderen.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionPlayerTag("Spieler, der übertragen werden soll."),
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "from_clan",
					Description: "Clan, aus dem das Mitglied übertragen werden soll.",
					Required:    true,
					Autocomplete: true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "to_clan",
					Description: "Clan, in den das Mitglied übertragen werden soll.",
					Required:    true,
					Autocomplete: true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "role",
					Description: "Rolle, die das Mitglied im neuen Clan haben soll.",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: models.RoleLeader.Format(), Value: models.RoleLeader.String()},
						{Name: models.RoleCoLeader.Format(), Value: models.RoleCoLeader.String()},
						{Name: models.RoleElder.Format(), Value: models.RoleElder.String()},
						{Name: models.RoleMember.Format(), Value: models.RoleMember.String()},
					},
				},
			},
		},
	}}
}
