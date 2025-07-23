package handlers

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"

	"bot/commands/components"
	"bot/commands/messages"
	"bot/commands/middleware"
	"bot/commands/repos"
	"bot/commands/util"
	"bot/commands/validation"
	"bot/store/postgres/models"
	"bot/types"
)

type IKickpointHandler interface {
	ClanKickpoints(s *discordgo.Session, i *discordgo.InteractionCreate)
	MemberKickpoints(s *discordgo.Session, i *discordgo.InteractionCreate)
	KickpointInfo(s *discordgo.Session, i *discordgo.InteractionCreate)
	ClanConfigModal(s *discordgo.Session, i *discordgo.InteractionCreate)
	ClanConfigModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate)
	KickpointHelp(s *discordgo.Session, i *discordgo.InteractionCreate)
	CreateKickpointModal(s *discordgo.Session, i *discordgo.InteractionCreate)
	CreateKickpointModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate)
	EditKickpointModal(s *discordgo.Session, i *discordgo.InteractionCreate)
	EditKickpointModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate)
	DeleteKickpoint(s *discordgo.Session, i *discordgo.InteractionCreate)
	AddKickpointReason(s *discordgo.Session, i *discordgo.InteractionCreate)
	EditKickpointReason(s *discordgo.Session, i *discordgo.InteractionCreate)
	DeleteKickpointReason(s *discordgo.Session, i *discordgo.InteractionCreate)
	NewKickpointLockHandler(lock bool) func(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate)
}

type KickpointHandler struct {
	kickpoints   repos.IKickpointsRepo
	reasons      repos.IKickpointReasonsRepo
	clans        repos.IClansRepo
	players      repos.IPlayersRepo
	members      repos.IMembersRepo
	clanSettings repos.IClanSettingsRepo
	memberStates repos.IMemberStatesRepo
	auth         middleware.AuthMiddleware
}

func NewKickpointHandler(kickpoints repos.IKickpointsRepo, reasons repos.IKickpointReasonsRepo, clans repos.IClansRepo, players repos.IPlayersRepo, members repos.IMembersRepo, clanSettings repos.IClanSettingsRepo, memberStates repos.IMemberStatesRepo, auth middleware.AuthMiddleware) IKickpointHandler {
	return &KickpointHandler{
		kickpoints:   kickpoints,
		reasons:      reasons,
		clans:        clans,
		players:      players,
		members:      members,
		clanSettings: clanSettings,
		memberStates: memberStates,
		auth:         auth,
	}
}

func (h *KickpointHandler) ClanKickpoints(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	if len(opts) != 1 {
		messages.SendInvalidInputErr(i, "Du musst einen Clan angeben.")
		return
	}

	clanTag := opts[0].StringValue()
	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleMember); err != nil {
		return
	}

	settings, err := h.clanSettings.ClanSettingsPreload(clanTag)
	if err != nil {
		messages.SendClanNotFound(i, clanTag)
		return
	}

	kickpoints, err := h.kickpoints.ActiveClanKickpoints(settings)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendEmbedResponse(i, messages.NewEmbed(
				"Keine Kickpunkte gefunden",
				fmt.Sprintf("In %s gibt es keine Mitglieder mit aktiven Kickpunkten.", settings.Clan.Name),
				messages.ColorRed,
			))
			return
		}
		messages.SendUnknownErr(i)
		return
	}

	messages.SendClanKickpoints(i, settings.Clan.Name, kickpoints)
}

func (h *KickpointHandler) MemberKickpoints(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	playerTag := util.StringOptionByName(PlayerTagOptionName, opts)
	if playerTag == "" {
		messages.SendInvalidInputErr(i, "Du musst ein Mitglied angeben.")
		return
	}

	// if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleMember); err != nil {
	// 	return
	// }

	var maxKickpoints, clanName string
	members, err := h.members.MembersByPlayerTag(playerTag)
	if err == nil && len(members) > 0 {
		clanTag := members[0].ClanTag

		settings, err := h.clanSettings.ClanSettingsPreload(clanTag)
		if err != nil {
			messages.SendClanNotFound(i, clanTag)
			return
		}

		maxKickpoints = strconv.Itoa(settings.MaxKickpoints)
		clanName = settings.Clan.Name
	} else {
		maxKickpoints = "???"
		clanName = "aktuell keinem Clan"
	}

	kickpointSum, err := h.kickpoints.KickpointSum(playerTag)
	if err != nil {
		kickpointSum = 0
	}

	kickpoints, err := h.kickpoints.ActiveMemberKickpoints(playerTag)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fields := make([]*discordgo.MessageEmbedField, 1)
			fields[0] = &discordgo.MessageEmbedField{
				Name:   "Gesamtanzahl (Vergangene und aktuelle Kickpunkte)",
				Value:  strconv.Itoa(kickpointSum),
				Inline: true,
			}

			embed := messages.NewFieldEmbed(
				"Keine aktiven Kickpunkte gefunden",
				"Dieses Mitglied hat keine aktiven Kickpunkte.",
				messages.ColorRed,
				fields,
			)

			messages.SendEmbedResponse(i, embed)
			return
		}
		messages.SendUnknownErr(i)
		return
	}

	messages.SendMemberKickpoints(i, kickpoints, kickpointSum, maxKickpoints, clanName)
}

func (h *KickpointHandler) KickpointInfo(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	if clanTag == "" {
		messages.SendInvalidInputErr(i, "Du musst einen Clan angeben.")
		return
	}

	settings, err := h.clanSettings.ClanSettingsPreload(clanTag)
	if err != nil {
		messages.SendClanNotFound(i, clanTag)
		return
	}

	reasons, err := h.reasons.KickpointReasons(clanTag)
	if err != nil {
		messages.SendUnknownErr(i)
		return
	}

	messages.SendKickpointInfo(i, settings, reasons)
}

func (h *KickpointHandler) ClanConfigModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)

	if clanTag == "" {
		messages.SendInvalidInputErr(i, "Es wurde kein Clan Tag angegeben.")
		return
	}

	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	settings, err := h.clanSettings.ClanSettingsPreload(clanTag)
	if err != nil {
		messages.SendClanNotFound(i, clanTag)
		return
	}

	if err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: util.BuildCustomID(i.ApplicationCommandData().Name, i.Interaction.Member.User.ID, clanTag),
			Title:    fmt.Sprintf("%s - Einstellungen", settings.Clan.Name),
			Components: components.GenModalComponents(
				components.ClanSettingMaxKickpoints(settings.MaxKickpoints),
				components.ClanSettingSeasonWins(settings.MinSeasonWins),
				components.ClanSettingExpiration(settings.KickpointsExpireAfterDays),
			),
		},
	}); err != nil {
		slog.Error("Error while responding to CreateKickpoint.", slog.Any("err", err))
	}
}

func (h *KickpointHandler) ClanConfigModalSubmit(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	if len(data.Components) != 3 {
		messages.SendEmbedResponse(i, messages.NewEmbed("Fehler", "Es wurden nicht alle Felder ausgefüllt.", messages.ColorRed))
		return
	}

	_, _, clanTag := util.ParseCustomID(data.CustomID)
	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	settings := &models.ClanSettings{
		ClanTag:                   clanTag,
		MaxKickpoints:             util.ParseIntModalInput(data.Components[0]),
		MinSeasonWins:             util.ParseIntModalInput(data.Components[1]),
		KickpointsExpireAfterDays: util.ParseIntModalInput(data.Components[2]),
	}

	if msg, ok := validation.ValidateClanSettings(settings); !ok {
		messages.SendInvalidInputErr(i, msg)
		return
	}

	if err := h.clanSettings.UpdateClanSettings(settings); err != nil {
		messages.SendUnknownErr(i)
		return
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Einstellungen aktualisiert",
		"Die Einstellungen wurden erfolgreich aktualisiert.",
		messages.ColorGreen,
	))
}

func (h *KickpointHandler) KickpointHelp(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	messages.SendKickpointHelp(i)
}

func (h *KickpointHandler) CreateKickpointModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	memberTag := util.StringOptionByName(MemberTagOptionName, opts)
	reasonName := util.StringOptionByName(ReasonOptionName, opts)

	if clanTag == "" || memberTag == "" || reasonName == "" {
		messages.SendInvalidInputErr(i, "Du musst einen Clan, ein Mitglied und einen Grund angeben.")
		return
	}

	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	isLocked, err := h.memberStates.IsKickpointLocked(memberTag, clanTag)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		messages.SendUnknownErr(i)
		return
	}
	if isLocked {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Mitglied gesperrt",
			"Dieses Mitglied kann momentan keine Kickpunkte erhalten, da es abgemeldet ist.",
			messages.ColorRed,
		))
		return
	}

	settings, err := h.clanSettings.ClanSettings(clanTag)
	if err != nil {
		messages.SendClanNotFound(i, clanTag)
		return
	}

	totalKickpoints, err := h.kickpoints.ActiveMemberKickpointsSum(memberTag)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		messages.SendUnknownErr(i)
		return
	}

	if totalKickpoints >= settings.MaxKickpoints {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Maximale Kickpunkte erreicht",
			fmt.Sprintf("Dieses Mitglied hat bereits %d/%d Kickpunkte und kann keine weiteren erhalten.", totalKickpoints, settings.MaxKickpoints),
			messages.ColorRed,
		))
		return
	}

	reason, err := h.reasons.KickpointReason(reasonName, clanTag)
	reasonLabel := reasonName
	if err == nil {
		reasonLabel = reason.Name
	}

	if err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: util.BuildCustomID(i.ApplicationCommandData().Name, i.Interaction.Member.User.ID, ""),
			Title:    "Kickpunkt hinzufügen",
			Components: components.GenModalComponents(
				components.KickpointReason(reasonLabel),
				components.KickpointDate(""),
				components.KickpointAmount(reason.Amount),
				components.Tag("Spieler Tag", memberTag, components.PlayerTagID),
				components.Tag("Clan Tag", clanTag, components.ClanTagID),
			),
		},
	}); err != nil {
		log.Printf("Error while responding to CreateKickpoint: %v", err)
	}
}

func (h *KickpointHandler) CreateKickpointModalSubmit(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	if len(data.Components) != 5 {
		messages.SendEmbedResponse(i, messages.NewEmbed("Fehler", "Es wurden nicht alle Felder ausgefüllt.", messages.ColorRed))
		return
	}

	clanTag := util.ParseStringModalInput(data.Components[4])
	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	date, err := util.ParseDateInput(data.Components[1])
	if err != nil {
		messages.SendInvalidInputErr(i, "Das eingegebene Datum ist ungültig. Es muss im DisplayString `DD.MM.YYYY` angegeben werden.")
		return
	}

	if date.After(time.Now()) {
		messages.SendInvalidInputErr(i, "Das eingegebene Datum liegt in der Zukunft.")
		return
	}

	settings, err := h.clanSettings.ClanSettingsPreload(clanTag)
	if err != nil {
		messages.SendClanNotFound(i, clanTag)
		return
	}

	mindate := util.KickpointMinDate(settings.KickpointsExpireAfterDays)
	if mindate.After(date) {
		messages.SendInvalidInputErr(i, fmt.Sprintf("Es können keine Kickpunkte vor %s vergeben werden, da diese schon abgelaufen wären.", util.FormatDate(mindate)))
		return
	}

	amount := util.ParseIntModalInput(data.Components[2])
	if amount <= 0 {
		messages.SendInvalidInputErr(i, "Die Anzahl an Kickpunkten muss mindestens 1 sein.")
		return
	}

	playerTag := util.ParseStringModalInput(data.Components[3])
	expiryDate := mindate.AddDate(0, 0, settings.KickpointsExpireAfterDays)

	_, userID, _ := util.ParseCustomID(data.CustomID)
	kickpoint := &models.Kickpoint{
		Description:        util.ParseStringModalInput(data.Components[0]),
		Date:               date,
		Amount:             amount,
		PlayerTag:          playerTag,
		ClanTag:            clanTag,
		CreatedByDiscordID: userID,
		UpdatedByDiscordID: userID,
		ExpiresAt:          expiryDate,
	}

	playerName, err := h.players.NameByTag(kickpoint.PlayerTag)
	if err != nil {
		messages.SendMemberNotFound(i, kickpoint.PlayerTag, kickpoint.ClanTag)
		return
	}

	if err = h.kickpoints.CreateKickpoint(kickpoint); err != nil {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Datenbankfehler",
			"Beim Speichern der Daten in der Datenbank ist ein Fehler aufgetreten. Dies liegt wahrscheinlich an der Eingabe ungültiger Daten.",
			messages.ColorRed,
		))
		return
	}

	messages.SendEmbedResponse(i, messages.NewFieldEmbed(
		fmt.Sprintf("Kickpunkt #%d erstellt", kickpoint.ID),
		"Der Kickpunkt wurde erstellt und gespeichert!",
		messages.ColorGreen,
		append([]*discordgo.MessageEmbedField{
			{Name: "Mitglied", Value: fmt.Sprintf("%s in %s", playerName, settings.Clan.Name)}},
			messages.DetailedKickpointFields(kickpoint)...,
		),
	))

	totalKickpoints, err := h.kickpoints.ActiveMemberKickpointsSum(kickpoint.PlayerTag)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	if err != nil {
		messages.SendChannelWarning(i.ChannelID, fmt.Sprintf("Bei der Überprüfung ob %s die maximale Anzahl an Kickpunkten erreicht hat, ist ein Fehler aufgetreten. Bitte überprüfe dies manuell.", playerName))
		return
	}

	if totalKickpoints >= settings.MaxKickpoints {
		messages.SendChannelWarning(i.ChannelID, fmt.Sprintf("%s hat die maximale Anzahl an Kickpunkten erreicht.", playerName))
	}
}

func (h *KickpointHandler) EditKickpointModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	id := util.UintOptionByName(IDOptionName, opts)
	if id == nil {
		messages.SendInvalidInputErr(i, "Du musst eine gültige Kickpunkt ID angeben.")
		return
	}

	kickpoint, err := h.kickpoints.KickpointByID(*id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendEmbedResponse(i, messages.NewEmbed(
				"Ungültige ID",
				"Es wurde kein Kickpunkt mit dieser ID gefunden.",
				messages.ColorRed,
			))
			return
		}

		messages.SendUnknownErr(i)
		return
	}

	if err = h.auth.AuthorizeInteraction(i, kickpoint.ClanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	if err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: util.BuildCustomID(i.ApplicationCommandData().Name, i.Interaction.Member.User.ID, strconv.Itoa(int(*id))),
			Title:    "Kickpunkt bearbeiten",
			Components: components.GenModalComponents(
				components.KickpointReason(kickpoint.Description),
				components.KickpointDate(util.FormatDate(kickpoint.Date)),
				components.KickpointAmount(kickpoint.Amount),
			),
		},
	}); err != nil {
		slog.Error("Error while responding to interaction.", slog.Any("err", err))
	}
}

func (h *KickpointHandler) EditKickpointModalSubmit(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	if len(data.Components) != 3 {
		messages.SendEmbedResponse(i, messages.NewEmbed("Fehler", "Es wurden nicht alle Felder ausgefüllt.", messages.ColorRed))
		return
	}

	date, err := util.ParseDateInput(data.Components[1])
	if err != nil {
		messages.SendInvalidInputErr(i, "Das eingegebene Datum ist ungültig. Es muss im DisplayString `DD.MM.YYYY` angegeben werden.")
		return
	}

	amount := util.ParseIntModalInput(data.Components[2])
	if amount <= 0 {
		messages.SendInvalidInputErr(i, "Die Anzahl der Kickpunkte muss größer als 0 sein.")
		return
	}

	_, userID, idStr := util.ParseCustomID(data.CustomID)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		messages.SendInvalidInputErr(i, "Es wurde eine ungültige ID angegeben.")
		return
	}

	prevKickpoint, err := h.kickpoints.KickpointByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendEmbedResponse(i, messages.NewEmbed(
				"Ungültige ID",
				"Es wurde kein Kickpunkt mit dieser ID gefunden.",
				messages.ColorRed,
			))
			return
		}

		messages.SendUnknownErr(i)
		return
	}

	if err = h.auth.AuthorizeInteraction(i, prevKickpoint.ClanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	// settings, err := h.clanSettings.ClanSettings(prevKickpoint.ClanTag)
	// if err != nil {
	// 	messages.SendClanNotFound(i, prevKickpoint.ClanTag)
	// 	return
	// }

	updatedKickpoint := &models.Kickpoint{
		ID:                 prevKickpoint.ID,
		Description:        util.ParseStringModalInput(data.Components[0]),
		Date:               date,
		Amount:             amount,
		UpdatedByDiscordID: userID,
	}

	updatedKickpoint, err = h.kickpoints.UpdateKickpoint(updatedKickpoint)
	if err != nil {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Datenbankfehler",
			"Beim Speichern der Daten in der Datenbank ist ein Fehler aufgetreten. Dies liegt wahrscheinlich an der Eingabe ungültiger Daten.",
			messages.ColorRed,
		))
		return
	}

	messages.SendEmbedResponse(i, messages.NewFieldEmbed(
		fmt.Sprintf("Kickpunkt #%d bearbeitet", updatedKickpoint.ID),
		"Der Kickpunkt wurde bearbeitet und gespeichert!",
		messages.ColorGreen,
		append([]*discordgo.MessageEmbedField{
			{Name: "Mitglied", Value: fmt.Sprintf("%s in %s", updatedKickpoint.Player.Name, updatedKickpoint.Clan.Name)}},
			messages.DetailedKickpointFields(updatedKickpoint)...,
		),
	))
}

func (h *KickpointHandler) DeleteKickpoint(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	id := util.UintOptionByName(IDOptionName, opts)
	if id == nil {
		messages.SendInvalidInputErr(i, "Du musst eine gültige Kickpunkt ID angeben.")
		return
	}

	kickpoint, err := h.kickpoints.KickpointByID(*id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendEmbedResponse(i, messages.NewEmbed(
				"Ungültige ID",
				"Es wurde kein Kickpunkt mit dieser ID gefunden.",
				messages.ColorRed,
			))
			return
		}

		messages.SendUnknownErr(i)
		return
	}

	if err = h.auth.AuthorizeInteraction(i, kickpoint.ClanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	if err = h.kickpoints.DeleteKickpoint(*id); err != nil {
		messages.SendUnknownErr(i)
		return
	}

	messages.SendEmbedResponse(i, messages.NewFieldEmbed(
		fmt.Sprintf("Kickpunkt #%d gelöscht", kickpoint.ID),
		fmt.Sprintf("Der Kickpunkt von %s in %s wurde gelöscht!", kickpoint.Player.Name, kickpoint.Clan.Name),
		messages.ColorGreen,
		messages.DetailedKickpointFields(kickpoint),
	))
}

func (h *KickpointHandler) NewKickpointLockHandler(lock bool) func(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		opts := i.ApplicationCommandData().Options
		clanTag := util.StringOptionByName(ClanTagOptionName, opts)
		memberTag := util.StringOptionByName(MemberTagOptionName, opts)
		if clanTag == "" || memberTag == "" {
			messages.SendInvalidInputErr(i, "Du musst einen Clan und ein Mitglied angeben.")
			return
		}

		if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
			return
		}

		if err := h.memberStates.UpdateKickpointLockStatus(memberTag, clanTag, lock); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				messages.SendEmbedResponse(i, messages.NewEmbed(
					"Ungültiger Spieler Tag",
					"Es wurde kein Mitglied mit diesem Spieler Tag gefunden.",
					messages.ColorRed,
				))
				return
			}
			messages.SendUnknownErr(i)
			return
		}

		title := "Mitglied angemeldet"
		desc := "Das Mitglied kann ab sofort wieder Kickpunkte erhalten."
		if lock {
			title = "Mitglied abgemeldet"
			desc = "Das Mitglied kann ab sofort keine Kickpunkte mehr erhalten."
		}

		messages.SendEmbedResponse(i, messages.NewEmbed(
			title,
			desc,
			messages.ColorGreen,
		))
	}
}

func (h *KickpointHandler) AddKickpointReason(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	reason := util.StringOptionByName(ReasonOptionName, opts)
	amount := util.IntOptionByName(AmountOptionName, opts)
	if clanTag == "" || reason == "" || amount == nil {
		messages.SendInvalidInputErr(i, "Du musst einen Clan, einen Grund und die Anzahl an Kickpunkten angeben.")
		return
	}

	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	if err := h.reasons.CreateKickpointReason(&models.KickpointReason{
		Name:    reason,
		Amount:  *amount,
		ClanTag: clanTag,
	}); err != nil {
		messages.SendUnknownErr(i)
		return
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Grund hinzugefügt",
		fmt.Sprintf("Der Grund `%s` mit %d Kickpunkten wurde erfolgreich hinzugefügt.", reason, *amount),
		messages.ColorGreen,
	))
}

func (h *KickpointHandler) EditKickpointReason(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	reason := util.StringOptionByName(ReasonOptionName, opts)
	amount := util.IntOptionByName(AmountOptionName, opts)
	if clanTag == "" || reason == "" || amount == nil {
		messages.SendInvalidInputErr(i, "Du musst einen Clan, einen Grund und die Anzahl an Kickpunkten angeben.")
		return
	}

	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	if err := h.reasons.UpdateKickpointReason(&models.KickpointReason{
		Name:    reason,
		Amount:  *amount,
		ClanTag: clanTag,
	}); err != nil {
		messages.SendUnknownErr(i)
		return
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Grund aktualisiert",
		fmt.Sprintf("Der Grund `%s` mit %d Kickpunkten wurde erfolgreich aktualisiert.", reason, *amount),
		messages.ColorGreen,
	))
}

func (h *KickpointHandler) DeleteKickpointReason(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	reason := util.StringOptionByName(ReasonOptionName, opts)
	if clanTag == "" || reason == "" {
		messages.SendInvalidInputErr(i, "Du musst einen Clan und einen Grund angeben.")
		return
	}

	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	if err := h.reasons.DeleteKickpointReason(reason, clanTag); err != nil {
		messages.SendUnknownErr(i)
		return
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Grund gelöscht",
		fmt.Sprintf("Der Grund `%s` wurde erfolgreich gelöscht.", reason),
		messages.ColorGreen,
	))
}

func (h *KickpointHandler) HandleAutocomplete(_ *discordgo.Session, i *discordgo.InteractionCreate) {
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
		case ReasonOptionName:
			h.autocompleteKickpointReason(i, opt.StringValue(), util.StringOptionByName(ClanTagOptionName, opts))
		case PlayerTagOptionName:
			autocompletePlayers(i, h.players, opt.StringValue())
		}
	}
}

func (h *KickpointHandler) autocompleteKickpointReason(i *discordgo.InteractionCreate, reason, clanTag string) {
	reasons, err := h.reasons.FindKickpointReasons(clanTag, reason)
	if err != nil {
		messages.SendAutoCompletion(i, nil)
		return
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, len(reasons))
	for index, r := range reasons {
		choices[index] = &discordgo.ApplicationCommandOptionChoice{
			Name:  r.Name,
			Value: r.Name,
		}
	}
	messages.SendAutoCompletion(i, choices)
}
