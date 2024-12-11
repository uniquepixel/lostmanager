package commands

import (
	"github.com/aaantiii/goclash"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"

	"bot/commands/handlers"
	"bot/commands/repos"
	"bot/commands/util"
	"bot/commands/validation"
	"bot/types"
)

func playerInteractionCommands(db *gorm.DB, client *goclash.Client) types.Commands[types.InteractionHandler] {
	handler := handlers.NewPlayerHandler(repos.NewPlayersRepo(db), client)

	return types.Commands[types.InteractionHandler]{{
		Handler: types.InteractionHandler{
			Main: handler.VerifyPlayer,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "verify",
			Description:  "Verknüpfe deinen Discord Account mit deinem COC-Account.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        handlers.PlayerTagOptionName,
					Description: "Spieler-Tag deines Clash of Clans Accounts.",
					Required:    true,
					MinLength:   util.IntPtr(validation.TagMinLength),
					MaxLength:   validation.TagMaxLength,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        handlers.ApiTokenOptionName,
					Description: "API Token deines Clash of Clans Accounts.",
					Required:    true,
					MinLength:   util.IntPtr(8),
					MaxLength:   8,
				},
			},
		},
	}, {
		Handler: types.InteractionHandler{
			Main:         handler.PingPlayer,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "pingplayer",
			Description:  "Pingt einen Spieler auf Discord durch seinen Namen oder Spieler Tag.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         handlers.PlayerTagOptionName,
					Description:  "Spieler-Tag des Spielers, der gepingt werden soll.",
					Required:     true,
					MinLength:    util.IntPtr(validation.TagMinLength),
					MaxLength:    validation.TagMaxLength,
					Autocomplete: true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        handlers.MessageOptionName,
					Description: "Nachricht, die an den Spieler gesendet werden soll.",
					Required:    true,
					MinLength:   util.IntPtr(1),
					MaxLength:   200,
				},
			},
		},
	}, {
		Handler: types.InteractionHandler{
			Main:         handler.SetNickname,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "setnick",
			Description:  "Ändert deinen Nicknamen zu deinem in-Game Namen, und optional einem benutzerdefinierten Alias.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         handlers.MyPlayerTagOptionName,
					Description:  "Spieler-Tag des Accounts, dessen Name als Nickname gesetzt werden soll.",
					Required:     true,
					MinLength:    util.IntPtr(validation.TagMinLength),
					MaxLength:    validation.TagMaxLength,
					Autocomplete: true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        handlers.AliasOptionName,
					Description: "Alias, der an deinen Namen angehängt werden soll. (z.B. Anti | [alias]).",
					MinLength:   util.IntPtr(1),
					MaxLength:   20,
				},
			},
		},
	}, {
		Handler: types.InteractionHandler{
			Main: handler.CheckReactions,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "checkreacts",
			Description:  "Pingt alle Mitglieder einer Rolle, welche auf eine Nachricht nicht reagiert haben.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        handlers.RoleOptionName,
					Description: "Rolle, welche auf Reaktionen geprüft werden soll.",
					Required:    true,
					MinLength:   util.IntPtr(1),
					MaxLength:   60,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        handlers.MessageIDOptionName,
					Description: "ID der Nachricht, welche auf Reaktionen geprüft werden soll.",
					MinLength:   util.IntPtr(19),
					MaxLength:   19,
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        handlers.EmojiOptionName,
					Description: "EmojiOptionName, mit welchem reagiert werden muss.",
					MinLength:   util.IntPtr(1),
					MaxLength:   50,
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        handlers.ChannelOptionName,
					Description: "Channel, in welchem die Nachricht zu finden ist.",
					Required:    false,
				},
			},
		},
	}}
}
