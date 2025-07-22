package commands

import (
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"

	"bot/commands/handlers"
	"bot/commands/middleware"
	"bot/commands/repos"
	"bot/commands/util"
	"bot/commands/validation"
	"bot/types"
)

func kickpointInteractionCommands(db *gorm.DB) types.Commands[types.InteractionHandler] {
	handler := handlers.NewKickpointHandler(
		repos.NewKickpointsRepo(db),
		repos.NewKickpointReasonsRepo(db),
		repos.NewClansRepo(db),
		repos.NewPlayersRepo(db),
		repos.NewClanSettingsRepo(db),
		repos.NewMemberStatesRepo(db),
		middleware.NewAuthMiddleware(repos.NewGuildsRepo(db), repos.NewClansRepo(db), repos.NewUsersRepo(db)),
	)

	return types.Commands[types.InteractionHandler]{{
		Handler: types.InteractionHandler{
			Main:         handler.ClanKickpoints,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpclan",
			Description:  "Alle Kickpunkte eines Clans anzeigen.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, dessen Kickpunkte angezeigt werden sollen."),
			},
		}}, {
		Handler: types.InteractionHandler{
			Main:         handler.MemberKickpoints,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpmember",
			Description:  "Alle Kickpunkte eines Mitglieds anzeigen.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				// optionClanTag("Clan, aus dem das Mitglied stammt."),
				// optionMemberTag("ClanMember, dessen Kickpunkte angezeigt werden sollen."),
				optionPlayerTag("Player, dessen Kickpunkte angezeigt werden sollen."),
				{
					Name:         handlers.ClanTagOptionName,
					Description:  "Clan, aus dem das Mitglied stammt.",
					Type:         discordgo.ApplicationCommandOptionString,
					Required:     false,
					Autocomplete: true,
					MinLength:    util.IntPtr(validation.TagMinLength),
					MaxLength:    validation.TagMaxLength,
				},
			},
		}}, {
		Handler: types.InteractionHandler{
			Main:         handler.KickpointInfo,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpinfo",
			Description:  "Übersicht wie viele Kickpunkte verschiedene Regelbrüche geben.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, dessen Übersicht angezeigt werden soll."),
			},
		}}, {
		Handler: types.InteractionHandler{
			Main:         handler.ClanConfigModal,
			Autocomplete: handler.HandleAutocomplete,
			ModalSubmit:  handler.ClanConfigModalSubmit,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "clanconfig",
			Description:  "Einstellungen eines Clans ändern.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, dessen Konfiguration geändert werden soll."),
			},
		}}, {
		Handler: types.InteractionHandler{
			Main:         handler.AddKickpointReason,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpaddreason",
			Description:  "Fügt einen Kickpunkte Grund für einen Clan hinzu.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, dessen Konfiguration geändert werden soll."),
				{
					Name:        handlers.ReasonOptionName,
					Description: "Grund, der hinzugefügt werden soll.",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
					MinLength:   util.IntPtr(8),
					MaxLength:   40,
				},
				{
					Name:        handlers.AmountOptionName,
					Description: "Anzahl der Kickpunkte, die dieser Grund gibt.",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    true,
					MinValue:    util.FloatPtr(1),
					MaxValue:    10,
				},
			},
		}}, {
		Handler: types.InteractionHandler{
			Main:         handler.DeleteKickpointReason,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpremovereason",
			Description:  "Entfernt einen Kickpunkte Grund von einem Clan.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, in dem der Grund gelöscht werden soll."),
				{
					Name:         handlers.ReasonOptionName,
					Description:  "Grund, der gelöscht werden soll.",
					Type:         discordgo.ApplicationCommandOptionString,
					Required:     true,
					Autocomplete: true,
					MinLength:    util.IntPtr(8),
					MaxLength:    40,
				},
			},
		}}, {
		Handler: types.InteractionHandler{
			Main:         handler.EditKickpointReason,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpeditreason",
			Description:  "Aktualisiert die Anzahl der Kickpunkte von einem Kickpunkte Grund.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan, in dem der Grund aktualisiert werden soll."),
				{
					Name:         handlers.ReasonOptionName,
					Description:  "Grund, dessen Kickpunkte Anzahl geändert werden soll.",
					Type:         discordgo.ApplicationCommandOptionString,
					Required:     true,
					Autocomplete: true,
					MinLength:    util.IntPtr(8),
					MaxLength:    40,
				},
				{
					Name:        handlers.AmountOptionName,
					Description: "Anzahl der Kickpunkte, die dieser Grund geben soll.",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    true,
					MinValue:    util.FloatPtr(1),
					MaxValue:    10,
				},
			},
		}}, {
		Handler: types.InteractionHandler{Main: handler.KickpointHelp},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kphelp",
			Description:  "Erklärung vom Kickpunkte System des Bots sowie den wichtigsten Befehlen.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
		}}, {
		Handler: types.InteractionHandler{
			Main:         handler.CreateKickpointModal,
			ModalSubmit:  handler.CreateKickpointModalSubmit,
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpadd",
			Description:  "Neuen Kickpunkt hinzufügen",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan aus dem das Mitglied stammt."),
				optionMemberTag("Mitglied, welches einen Kickpunkt erhält."),
				{
					Name:         handlers.ReasonOptionName,
					Description:  "Grund des Kickpunktes (wird für die Anzahl der Kickpunkte benötigt).",
					Type:         discordgo.ApplicationCommandOptionString,
					Required:     true,
					Autocomplete: true,
				}},
		}}, {
		Handler: types.InteractionHandler{
			Main:        handler.EditKickpointModal,
			ModalSubmit: handler.EditKickpointModalSubmit,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpedit",
			Description:  "Bestehenden Kickpunkt bearbeiten",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{{
				Name:        "id",
				Description: "ID des Kickpunktes, den du bearbeiten möchtest.",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
				MinValue:    util.FloatPtr(1),
			}},
		}}, {
		Handler: types.InteractionHandler{Main: handler.DeleteKickpoint},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpremove",
			Description:  "Bestehenden Kickpunkt löschen (kann nicht rückgängig gemacht werden)",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{{
				Name:        "id",
				Description: "ID des Kickpunktes, den du löschen möchtest.",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    true,
				MinValue:    util.FloatPtr(1),
			}},
		}}, {
		Handler: types.InteractionHandler{
			Main:         handler.NewKickpointLockHandler(true),
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kplock",
			Description:  "Mitglied abmelden, sodass es keine Kickpunkte mehr erhalten kann.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan aus dem das Mitglied stammt."),
				optionMemberTag("Mitglied, welches abgemeldet werden soll."),
			},
		}}, {
		Handler: types.InteractionHandler{
			Main:         handler.NewKickpointLockHandler(false),
			Autocomplete: handler.HandleAutocomplete,
		},
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:         "kpunlock",
			Description:  "Mitglied anmelden, sodass es wieder Kickpunkte erhalten kann.",
			Type:         discordgo.ChatApplicationCommand,
			DMPermission: util.BoolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				optionClanTag("Clan aus dem das Mitglied stammt."),
				optionMemberTag("Mitglied, welches wieder angemeldets werden soll."),
			},
		}},
	}
}
