package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"github.com/aaantiii/goclash"
	"github.com/bwmarrin/discordgo"
	cmap "github.com/orcaman/concurrent-map/v2"
	"gorm.io/gorm"

	"bot/commands/messages"
	"bot/commands/middleware"
	"bot/commands/repos"
	"bot/commands/util"
	"bot/commands/validation"
	"bot/store/postgres/models"
	"bot/types"
)

type IClanHandler interface {
	ClanStats(s *discordgo.Session, i *discordgo.InteractionCreate)
	RaidPing(s *discordgo.Session, i *discordgo.InteractionCreate)
	EventInfo(s *discordgo.Session, i *discordgo.InteractionCreate)
	CreateEvent(s *discordgo.Session, i *discordgo.InteractionCreate)
	DeleteEvent(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate)
	CWDonator(s *discordgo.Session, i *discordgo.InteractionCreate)
}

type ClanHandler struct {
	clans          repos.IClansRepo
	members        repos.IMembersRepo
	events         repos.IClanEventsRepo
	clashClient    *goclash.Client
	auth           middleware.AuthMiddleware
	eventCancelers cmap.ConcurrentMap[string, context.CancelFunc]
}

func NewClanHandler(clans repos.IClansRepo, members repos.IMembersRepo, events repos.IClanEventsRepo, auth middleware.AuthMiddleware, clashClient *goclash.Client) IClanHandler {
	h := &ClanHandler{
		clans:          clans,
		members:        members,
		events:         events,
		clashClient:    clashClient,
		auth:           auth,
		eventCancelers: cmap.New[context.CancelFunc](),
	}

	activeEvents, err := h.events.AllActiveClanEvents()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Error("Error while fetching active clan events", slog.Any("err", err))
	}
	for _, event := range activeEvents {
		go h.watchEvent(event)
	}

	return h
}

func (h *ClanHandler) CWDonator(s *discordgo.Session, i *discordgo.InteractionCreate) {
	slog.Info("Received interaction", slog.Any("interactionID", i.ID), slog.Any("interactionType", i.Type))

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		slog.Error("Failed to send deferred response", slog.Any("err", err), slog.Any("interactionID", i.ID))
		return
	}

	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)

	if clanTag == "" {
		if err = messages.CreateAndEditEmbed(s, i, "Ungültige Eingabe", "Bitte gib einen Clan Tag an.", messages.ColorRed); err != nil {
			slog.Error("Error while editing embed", slog.Any("err", err))
		}
		return
	}

	if err = h.auth.AuthorizeInteractionWithMessageEditNoClanNeeded(s, i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	clan, err := h.clashClient.GetClan(clanTag)
	if err != nil {
		err = messages.CreateAndEditEmbed(s, i, "Clan nicht gefunden", fmt.Sprintf("Der Clan mit dem Tag `%s` konnte nicht gefunden werden. Stelle sicher, dass du den Clan aus der Liste ausgewählt hast, oder direkt einen Clan Tag eingegeben hast.", clanTag), messages.ColorRed)
		if err != nil {
			slog.Error("Failed to edit message.", slog.Any("err", err))
		}
		return
	}
	clanName := clan.Name

	clanWar, err := h.clashClient.GetCurrentClanWar(clanTag)
	if err != nil {
		if err = messages.CreateAndEditEmbed(s, i, "Fehler", "Beim Abrufen der aktuellen Clan Kriegsdaten ist ein Fehler aufgetreten.", messages.ColorRed); err != nil {
			slog.Error("Failed to edit message.", slog.Any("err", err))
		}
		return
	}

	clanWarPlayerTags := make([]string, len(clanWar.Clan.Members))
	for index, member := range clanWar.Clan.Members {
		clanWarPlayerTags[index] = member.Tag
	}

	var members []*models.ClanMember
	_, err = h.clans.ClanByTag(clanTag)
	if err == nil {
		members, err = h.members.MembersByClanTag(clanTag)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err = messages.CreateAndEditEmbed(s, i, "Clan hat keine Mitglieder", fmt.Sprintf("Der Clan '%s' hat keine Mitglieder.", clanName), messages.ColorRed); err != nil {
					slog.Error("Failed to edit message.", slog.Any("err", err))
				}
				return
			}

			if err = messages.CreateAndEditEmbed(s, i, "Unbekannter Fehler", "Es ist ein unbekannter Fehler aufgetreten.", messages.ColorRed); err != nil {
				slog.Error("Failed to edit message.", slog.Any("err", err))
			}
			return
		}
	} else {
		for i := 0; i < len(clanWarPlayerTags); i++ {
			member, _ := h.members.MembersByPlayerTag(clanWarPlayerTags[i])
			if member != nil {
				members = append(members, member...)
			}
		}
	}

	clanPlayers, err := h.clashClient.GetPlayersWithError(clanWarPlayerTags...)
	if err != nil {
		if err = messages.CreateAndEditEmbed(s, i, "Fehler", "Beim Abrufen der Kriegs Teilnehmer ist ein Fehler aufgetreten.", messages.ColorRed); err != nil {
			slog.Error("Failed to edit message.", slog.Any("err", err))
		}
		return
	}

	clanPlayerByTag := make(map[string]*goclash.Player, len(clanPlayers))
	for _, clanPlayer := range clanPlayers {
		clanPlayerByTag[clanPlayer.Tag] = clanPlayer
	}

	if _, err = util.Session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			messages.NewEmbed("CW Spender", "Folgende Mitglieder wurden zufällig als Spender ausgewählt:", messages.ColorAqua),
		},
	}); err != nil {
		slog.Error("Failed to edit message.", slog.Any("err", err))
	}

	if _, err = util.Session.ChannelMessageSend(i.ChannelID, messages.CWDonatorPing(members, clanWar.Clan.Members, clanPlayerByTag)); err != nil {
		slog.Error("Error sending message", slog.Any("err", err))
	}
}

func (h *ClanHandler) ClanStats(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	statistic := util.StringOptionByName(StatisticOptionName, opts)

	if clanTag == "" || statistic == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Clan und eine Statistik an.")
		return
	}

	compStat := util.ComparableStatisticByName(statistic)
	if compStat == nil {
		messages.SendInvalidInputErr(i, "Es wurde keine gültige Statistik angegeben.")
		return
	}

	clan, err := h.clans.ClanByTagPreload(clanTag)
	if err != nil {
		messages.SendClanNotFound(i, clanTag)
		return
	}

	players, err := h.clashClient.GetPlayersWithError(clan.ClanMembers.Tags()...)
	if err != nil {
		messages.SendCocApiErr(i, err)
		return
	}

	statValues, err := util.StatisticValueFromPlayers(players, compStat)
	if err != nil {
		messages.SendErr(i, "Es wurde eine ungültige Statistik angegeben.")
		return
	}

	var playerStats []*types.PlayerStatistic
	for index, player := range players {
		if player == nil {
			continue
		}
		playerStats = append(playerStats, &types.PlayerStatistic{
			Tag:   player.Tag,
			Name:  player.Name,
			Value: statValues[index],
		})
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		fmt.Sprintf("%s (%s)", compStat.DisplayName, clan.Name),
		messages.PlayerLeaderboardTable(playerStats),
		messages.ColorAqua,
	))
}

func (h *ClanHandler) RaidPing(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)

	if clanTag == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Clan und ein Mitglied an.")
		return
	}
	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	clanName, err := h.clans.ClanNameByTag(clanTag)
	if err != nil {
		messages.SendClanNotFound(i, clanTag)
		return
	}

	members, err := h.members.MembersByClanTag(clanTag)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendClanHasNoMembers(i, clanName)
			return
		}
		messages.SendUnknownErr(i)
		return
	}

	raid, err := h.clashClient.GetClanCapitalRaidSeasons(clanTag, &goclash.PagingParams{
		Limit: 1,
	})
	if err != nil {
		messages.SendCocApiErr(i, err)
		return
	}

	if len(raid.Items) == 0 {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Keine Raids gefunden",
			"Es wurden keine Raids gefunden.",
			messages.ColorRed,
		))
		return
	}

	messages.SendRaidPing(i, members, raid.Items[0])
}

func (h *ClanHandler) EventInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	eventID := util.UintOptionByName(IDOptionName, opts)

	if eventID == nil {
		messages.SendInvalidInputErr(i, "Bitte gib eine Event ID an.")
		return
	}

	event, err := h.events.ClanEventByID(*eventID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendErr(i, fmt.Sprintf("Es wurde kein Clan Event mit der ID %d gefunden.", *eventID))
			return
		}
		messages.SendUnknownErr(i)
		return
	}

	if event.StartsAt.After(time.Now()) {
		messages.SendEmbedResponse(i, messages.NewFieldEmbed(
			fmt.Sprintf("Event #%d", event.ID),
			fmt.Sprintf("Das Event startet in %s.", util.FormatDuration(time.Until(event.StartsAt))),
			messages.ColorAqua,
			messages.EventEmbedFields(event, nil),
		))
		return
	}

	membersAtStart, err := h.events.ClanEventMembers(event.ID, event.StartsAt)
	if err != nil {
		messages.SendUnknownErr(i)
		return
	}

	var currentMembers []*models.ClanEventMember
	if event.WinnerPlayerTag == nil {
		tags := make([]string, len(membersAtStart))
		for index, member := range membersAtStart {
			tags[index] = member.PlayerTag
		}

		currentMembers, err = h.fetchEventMembers(event, time.Now(), tags)
		if err != nil {
			messages.SendCocApiErr(i, err)
			return
		}
	} else {
		currentMembers, err = h.events.ClanEventMembers(event.ID, event.EndsAt)
		if err != nil {
			messages.SendUnknownErr(i)
			return
		}
	}

	memberAtStartByTag := make(map[string]*models.ClanEventMember, len(membersAtStart))
	for _, member := range membersAtStart {
		memberAtStartByTag[member.PlayerTag] = member
	}

	stats := make(types.PlayerStatistics, 0, len(currentMembers))
	for _, member := range currentMembers {
		memberAtStart, ok := memberAtStartByTag[member.PlayerTag]
		if !ok {
			continue
		}
		stats = append(stats, &types.PlayerStatistic{
			Tag:   member.PlayerTag,
			Name:  member.Name,
			Value: member.Value - memberAtStart.Value,
		})
	}

	messages.SendEmbedResponse(i, messages.NewFieldEmbed(
		fmt.Sprintf("Event #%d - Übersicht", event.ID),
		messages.PlayerLeaderboardTable(stats),
		messages.ColorAqua,
		messages.EventEmbedFields(event, stats),
	))
}

func (h *ClanHandler) CreateEvent(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	clanTag := util.StringOptionByName(ClanTagOptionName, opts)
	statName := util.StringOptionByName(StatisticOptionName, opts)
	if clanTag == "" || statName == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Clan und eine Statistik an.")
		return
	}
	if err := h.auth.AuthorizeInteraction(i, clanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	compStat := util.ComparableStatisticByName(statName)
	if compStat == nil {
		messages.SendInvalidInputErr(i, "Es wurde keine gültige Statistik angegeben.")
		return
	}

	startsAt, err := util.DateTimeOptionByName(StartsAtOptionName, opts)
	if err != nil {
		messages.SendInvalidDateTimeFormat(i, StartsAtOptionName)
		return
	}
	endsAt, err := util.DateTimeOptionByName(EndsAtOptionName, opts)
	if err != nil {
		messages.SendInvalidDateTimeFormat(i, EndsAtOptionName)
		return
	}
	if msg, ok := validation.ValidateEventDates(startsAt, endsAt); !ok {
		messages.SendInvalidInputErr(i, msg)
		return
	}

	clanName, err := h.clans.ClanNameByTag(clanTag)
	if err != nil {
		messages.SendClanNotFound(i, clanTag)
		return
	}

	id, err := h.events.CreateClanEvent(&models.ClanEvent{
		ClanTag:   clanTag,
		StatName:  statName,
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		ChannelID: i.ChannelID,
	})
	if err != nil {
		messages.SendUnknownErr(i)
		return
	}

	event, err := h.events.ClanEventByID(id)
	if err != nil {
		messages.SendUnknownErr(i)
		return
	}
	go h.watchEvent(event)

	messages.SendEmbedResponse(i, messages.NewFieldEmbed(
		"Clan Event erstellt",
		fmt.Sprintf("Das Clan Event für %s wurde erfolgreich erstellt.", clanName),
		messages.ColorGreen,
		messages.EventEmbedFields(event, nil),
	))
}

func (h *ClanHandler) DeleteEvent(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	eventID := util.UintOptionByName(IDOptionName, opts)

	if eventID == nil {
		messages.SendInvalidInputErr(i, "Bitte gib eine Event ID an.")
		return
	}

	event, err := h.events.ClanEventByID(*eventID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendErr(i, fmt.Sprintf("Es wurde kein Clan Event mit der ID %d gefunden.", eventID))
			return
		}
		messages.SendUnknownErr(i)
		return
	}

	if event.WinnerPlayerTag != nil {
		messages.SendErr(i, "Das Event ist bereits beendet und kann nicht mehr gelöscht werden.")
		return
	}

	if err = h.auth.AuthorizeInteraction(i, event.ClanTag, types.AuthRoleCoLeader); err != nil {
		return
	}

	if err = h.events.DeleteClanEvent(*eventID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			messages.SendErr(i, fmt.Sprintf("Es wurde kein Clan Event mit der ID %d gefunden.", eventID))
			return
		}
		messages.SendUnknownErr(i)
		return
	}
	if cancel, ok := h.eventCancelers.Pop(strconv.Itoa(int(*eventID))); ok {
		cancel()
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Clan Event gelöscht",
		fmt.Sprintf("Das Clan Event mit der ID %d wurde gelöscht.", event.ID),
		messages.ColorGreen,
	))
}

func (h *ClanHandler) watchEvent(event *models.ClanEvent) {
	if event.WinnerPlayerTag != nil {
		return
	}
	if event.EndsAt.Before(time.Now()) {
		if err := h.onEventEnd(event); err != nil {
			log.Printf("Error while ending event: %v", err)
		}
		return
	}
	if event.StartsAt.Before(time.Now()) {
		if _, err := h.events.ClanEventMembers(event.ID, event.StartsAt); err != nil {
			if err := h.events.DeleteClanEvent(event.ID); err != nil {
				log.Printf("watchEvent: error while deleting event %v:", err)
			}
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.eventCancelers.Set(strconv.Itoa(int(event.ID)), cancel)
	defer h.eventCancelers.Remove(strconv.Itoa(int(event.ID)))

	startsAtTimer := time.NewTimer(event.StartsAt.Sub(time.Now()))
	endsAtTimer := time.NewTimer(event.EndsAt.Sub(time.Now()))
	for {
		select {
		case <-startsAtTimer.C:
			startsAtTimer.Stop()
			if err := h.onEventStart(event); err != nil {
				messages.SendChannelEmbed(event.ChannelID, messages.NewEmbed(
					"Fehler",
					"Beim Starten des Events ist ein unerwarteter Fehler aufgetreten.",
					messages.ColorRed,
				))
				return
			}
		case <-endsAtTimer.C:
			if err := h.onEventEnd(event); err != nil {
				messages.SendChannelEmbed(event.ChannelID, messages.NewEmbed(
					"Fehler",
					"Beim Beenden des Events ist ein unerwarteter Fehler aufgetreten.",
					messages.ColorRed,
				))
				log.Printf("Error while ending event: %v", err)
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func (h *ClanHandler) fetchEventMembers(event *models.ClanEvent, timestamp time.Time, tags []string) ([]*models.ClanEventMember, error) {
	if len(tags) == 0 {
		return nil, errors.New("no tags provided")
	}

	players, err := h.clashClient.GetPlayersWithError(tags...)
	if err != nil {
		return nil, err
	}

	values, err := util.StatisticValueFromPlayers(players, util.ComparableStatisticByName(event.StatName))
	if err != nil {
		return nil, err
	}

	members := make([]*models.ClanEventMember, len(players))
	for i, player := range players {
		members[i] = &models.ClanEventMember{
			ClanEventID: event.ID,
			PlayerTag:   player.Tag,
			ClanTag:     event.ClanTag,
			Timestamp:   timestamp,
			Value:       values[i],
			Name:        player.Name,
		}
	}

	return members, nil
}

func (h *ClanHandler) onEventStart(event *models.ClanEvent) error {
	clanMembers, err := h.members.MembersByClanTag(event.ClanTag)
	if err != nil {
		return err
	}

	members, err := h.fetchEventMembers(event, event.StartsAt, clanMembers.Tags())
	if err != nil {
		return err
	}

	if err = h.events.CreateClanEventMembers(members); err != nil {
		return err
	}

	if _, err = util.Session.ChannelMessageSendEmbed(event.ChannelID, messages.NewFieldEmbed(
		"Ein Event startet!",
		fmt.Sprintf("Ein Event in %s hat gerade begonnen!\nZeit bis zum Ende: %s", event.Clan.Name, util.FormatDuration(time.Until(event.EndsAt))),
		messages.ColorGreen,
		messages.EventEmbedFields(event, nil),
	)); err != nil {
		log.Printf("Error while sending event start message: %v\nThis error doesn't cancel the event, it is still tried to be ended at %s.", err, event.EndsAt.Round(time.Minute).String())
	}
	return nil
}

func (h *ClanHandler) onEventEnd(event *models.ClanEvent) error {
	if event.WinnerPlayerTag != nil {
		return errors.New("onEventEnd: event has already ended")
	}

	eventMembersAtStart, err := h.events.ClanEventMembers(event.ID, event.StartsAt)
	if err != nil {
		return err
	}

	tags := make([]string, len(eventMembersAtStart))
	eventMemberByTag := make(map[string]*models.ClanEventMember, len(eventMembersAtStart))
	for i, member := range eventMembersAtStart {
		tags[i] = member.PlayerTag
		eventMemberByTag[member.PlayerTag] = member
	}

	eventMembersAtEnd, err := h.fetchEventMembers(event, event.EndsAt, tags)
	if err != nil {
		return err
	}
	if err = h.events.CreateClanEventMembers(eventMembersAtEnd); err != nil {
		return err
	}

	finalStats := make(types.PlayerStatistics, 0, len(eventMembersAtEnd))
	for _, memberAtEnd := range eventMembersAtEnd {
		memberAtStart, ok := eventMemberByTag[memberAtEnd.PlayerTag]
		if !ok {
			continue
		}
		finalStats = append(finalStats, &types.PlayerStatistic{
			Tag:   memberAtEnd.PlayerTag,
			Value: memberAtEnd.Value - memberAtStart.Value,
			Name:  memberAtEnd.Name,
		})
	}

	sort.SliceStable(finalStats, func(i, j int) bool {
		return finalStats[i].Value > finalStats[j].Value
	})
	event.WinnerPlayerTag = &finalStats[0].Tag
	if err = h.events.UpdateClanEvent(event); err != nil {
		return err
	}

	if err = util.Session.ChannelTyping(event.ChannelID); err != nil {
		log.Printf("onEventEnd: failed to send typing indicator: %v", err)
	}
	time.Sleep(time.Second * 10)
	_, err = util.Session.ChannelMessageSendEmbed(event.ChannelID, messages.NewFieldEmbed(
		"Event beendet",
		messages.PlayerLeaderboardTable(finalStats),
		messages.ColorAqua,
		messages.EventEmbedFields(event, finalStats),
	))
	return err
}

func (h *ClanHandler) HandleAutocomplete(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	for _, opt := range i.ApplicationCommandData().Options {
		if !opt.Focused {
			continue
		}

		switch opt.Name {
		case ClanTagOptionName:
			autocompleteClans(i, h.clans, opt.StringValue())
		}
	}
}
