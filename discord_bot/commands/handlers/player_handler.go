package handlers

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/aaantiii/goclash"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"

	"bot/commands/messages"
	"bot/commands/repos"
	"bot/commands/util"
	"bot/env"
	"bot/store/postgres/models"
)

type IPlayerHandler interface {
	VerifyPlayer(s *discordgo.Session, i *discordgo.InteractionCreate)
	PingPlayer(s *discordgo.Session, i *discordgo.InteractionCreate)
	HandleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate)
	SetNickname(s *discordgo.Session, i *discordgo.InteractionCreate)
	CheckReactions(s *discordgo.Session, i *discordgo.InteractionCreate)
}

type PlayerHandler struct {
	players     repos.IPlayersRepo
	clashClient *goclash.Client
}

const cocVerificationStatusOK = "ok"

func NewPlayerHandler(players repos.IPlayersRepo, clashClient *goclash.Client) IPlayerHandler {
	return &PlayerHandler{
		players:     players,
		clashClient: clashClient,
	}
}

func (h *PlayerHandler) VerifyPlayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	go util.DeleteInteractionResponseWithTimeout(s, i.Interaction, time.Minute*2)

	playerTag := util.StringOptionByName(PlayerTagOptionName, opts)
	apiToken := util.StringOptionByName(ApiTokenOptionName, opts)
	if playerTag == "" || apiToken == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Spieler-Tag und einen API-Token an.")
		return
	}

	if !strings.HasPrefix(playerTag, "#") {
		messages.SendInvalidInputErr(i, "Bitte gib einen gültigen Spieler-Tag an.")
		return
	}

	verification, err := h.clashClient.VerifyPlayer(playerTag, apiToken)
	if err != nil {
		var apiErr *goclash.ClientError
		if !errors.As(err, &apiErr) {
			messages.SendUnknownErr(i)
			return
		}

		messages.SendInvalidInputErr(i, fmt.Sprintf("Bei der Anfrage zur Clash of Clans-API ist ein Fehler aufgetreten (Fehlercode %d).", apiErr.Status))
		return
	}

	if verification.Status != cocVerificationStatusOK {
		messages.SendInvalidInputErr(i, "Dein API-Token oder Spieler Tag ist ungültig. Bitte versuche es erneut mit gültigen Eingaben.")
		return
	}

	player, err := h.clashClient.GetPlayer(playerTag)
	if err != nil {
		messages.SendCocApiErr(i, err)
		return
	}

	if _, err = h.players.PlayerByTagAndDiscordID(playerTag, i.Member.User.ID); err == nil {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Account bereits verifiziert",
			fmt.Sprintf("%s, dein Clash of Clans Account %s (%s) ist bereits verifiziert und mit deinem Discord Account verknüpft!", i.Member.Mention(), player.Name, player.Tag),
			messages.ColorRed,
		))
		return
	}

	existingPlayers, err := h.players.PlayersByDiscordID(i.Member.User.ID)
	if (err == nil || errors.Is(err, gorm.ErrRecordNotFound)) && len(existingPlayers) == 0 {
		if _, err = s.GuildMemberEdit(i.GuildID, i.Member.User.ID, &discordgo.GuildMemberParams{
			Nick: player.Name,
		}); err != nil {
			messages.SendEmbedResponse(i, messages.NewEmbed(
				"Fehler",
				"Der Bot konnte deinen Discord Namen nicht zu deinem Clash of Clans Namen ändern. Dies liegt wahrscheinlich an fehlenden Berechtigungen des Bots.",
				messages.ColorRed,
			))
			return
		}
	}

	if err = h.players.CreateOrUpdatePlayer(&models.Player{
		CocTag:    player.Tag,
		Name:      player.Name,
		DiscordID: i.Member.User.ID,
	}); err != nil {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Datenbankfehler",
			"Beim Speichern deines Accounts in der Datenbank ist ein Fehler aufgetreten. Bitte versuche es erneut.",
			messages.ColorRed,
		))
		return
	}

	if err = s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, env.DISCORD_VERIFIED_ROLE_ID.Value()); err != nil {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Erfolgreich verifiziert",
			fmt.Sprintf(
				"%s, dein Clash of Clans Account %s (%s) wurde erfolgreich verifiziert und mit deinem Discord Account verknüpft!\n**Achtung**: Der Bot konnte dir die Verified-Rolle nicht automatisch zuweisen. Bitte frage einen Vize, ob er sie dir manuell geben kann.",
				i.Member.Mention(),
				player.Name,
				player.Tag),
			messages.ColorGreen,
		))
		return
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Erfolgreich verifiziert",
		fmt.Sprintf("%s, dein Clash of Clans Account %s (%s) wurde erfolgreich verifiziert und mit deinem Discord Account verknüpft!", i.Member.Mention(), player.Name, player.Tag),
		messages.ColorGreen,
	))
}

func (h *PlayerHandler) PingPlayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	playerTag := util.StringOptionByName(PlayerTagOptionName, opts)
	msg := util.StringOptionByName(MessageOptionName, opts)

	if playerTag == "" || msg == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Spieler-Tag und eine Nachricht ein.")
		return
	}

	if !strings.HasPrefix(playerTag, "#") {
		messages.SendInvalidInputErr(i, "Bitte gib einen gültigen Spieler-Tag an.")
		return
	}

	player, err := h.players.PlayerByTag(playerTag)
	if err != nil {
		messages.SendErr(i, "Es wurde kein Spieler mit dem angegebenen Spieler-Tag gefunden.")
		return
	}

	if _, err = s.ChannelMessageSend(i.ChannelID, fmt.Sprintf("%s, du wurdest von %s gepingt:\n%s", util.MentionUserID(player.DiscordID), i.Member.Mention(), msg)); err != nil {
		log.Printf("PingPlayer failed to send message: %v", err)
		messages.SendErr(i, "Beim Senden des Pings ist ein Fehler aufgetreten.")
		return
	}

	messages.SendEmptyResponse(i)
}

func (h *PlayerHandler) SetNickname(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	playerTag := util.StringOptionByName(MyPlayerTagOptionName, opts)
	alias := util.StringOptionByName(AliasOptionName, opts)

	if playerTag == "" {
		messages.SendInvalidInputErr(i, "Bitte gib einen Spieler-Tag an.")
		return
	}
	if !strings.HasPrefix(playerTag, "#") {
		playerTag = "#" + playerTag
	}

	player, err := h.players.PlayerByTag(playerTag)
	if err != nil {
		messages.SendErr(i, "Es wurde kein Spieler mit dem angegebenen Spieler-Tag gefunden.")
		return
	}

	if player.DiscordID != i.Member.User.ID {
		messages.SendErr(i, fmt.Sprintf("Der Account %s ist nicht mit deinem Discord Account verknüpft.", playerTag))
		return
	}

	nick := player.Name
	if alias != "" {
		nick += fmt.Sprintf(" | %s", alias)
	}
	if _, err = s.GuildMemberEdit(i.GuildID, player.DiscordID, &discordgo.GuildMemberParams{
		Nick: nick,
	}); err != nil {
		messages.SendErr(i, "Beim Ändern deines Nicknamen ist ein Fehler aufgetreten.")
		return
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Erfolgreich geändert",
		fmt.Sprintf("Dein Nickname wurde zu %s geändert.", nick),
		messages.ColorGreen,
	))
}

func (h *PlayerHandler) CheckReactions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	role, roleErr := util.RoleOptionByName(RoleOptionName, i.GuildID, opts) // role to check reactions for
	messageID := util.StringOptionByName(MessageIDOptionName, opts)         // message to check reactions for
	emoji, emojiErr := util.EmojiOptionByName(EmojiOptionName, opts)        // emoji to check reactions for
	channel, _ := util.ChannelOptionByName(ChannelOptionName, opts)         // channel to check reactions for

	if roleErr != nil || emojiErr != nil {
		messages.SendInvalidInputErr(i, "Bitte gib eine Rolle, die Nachricht-ID und die Emoji-ID an.")
		return
	}

	var channelID string
	if channel != nil {
		channelID = channel.ID
	}

	if channelID == "" {
		channelID = i.ChannelID
	}

	slog.Debug("data from CheckReactions", slog.Any("role", role), slog.String("messageID", messageID), slog.Any("emoji", emoji))

	if messageID == "" {
		messages.SendInvalidInputErr(i, "Bitte gib eine Rolle, die Nachricht-ID und die Emoji-ID an.")
		return
	}

	usersReacted, err := s.MessageReactions(channelID, messageID, emoji.GlobalID, 100, "", "")
	if err != nil {
		slog.Debug("Error while getting reactions for message.", slog.Any("err", err))
		messages.SendErr(i, "Die Reaktionen auf die Nachricht konnten nicht abgerufen werden. Dies kann daran liegen, dass niemand mit diesem Emoji auf die Nachricht reagiert hat, oder dass die Nachricht in einem anderen Channel geschrieben wurde.")
		return
	}
	var userIDsReacted []string
	for _, userReacted := range usersReacted {
		userIDsReacted = append(userIDsReacted, userReacted.ID)
	}

	members, err := s.GuildMembers(i.GuildID, "", 1000)
	if err != nil {
		messages.SendErr(i, "Die Mitglieder des Discord Servers konnten nicht abgerufen werden.")
		return
	}

	var missingUserIDs []string
	for _, member := range members {
		if !slices.Contains(member.Roles, role.ID) {
			continue
		}
		if !slices.Contains(userIDsReacted, member.User.ID) {
			missingUserIDs = append(missingUserIDs, member.User.ID)
		}
	}

	if len(missingUserIDs) == 0 {
		messages.SendEmbedResponse(i, messages.NewEmbed(
			"Alle Mitglieder haben reagiert",
			fmt.Sprintf(
				"Alle Mitglieder der Rolle %s haben auf die Nachricht %s reagiert.",
				util.MentionRole(role.ID),
				util.CreateMessageURL(i.GuildID, channelID, messageID),
			),
			messages.ColorGreen,
		))
		return
	}

	mentions := make([]string, len(missingUserIDs))
	for index, userID := range missingUserIDs {
		mentions[index] = util.MentionUserID(userID)
	}

	messages.SendEmbedResponse(i, messages.NewEmbed(
		"Mitglieder ohne Reaktion",
		fmt.Sprintf(
			"Folgende Mitglieder der Rolle %s haben noch nicht mit %s auf die Nachricht %s reagiert:",
			util.MentionRole(role.ID),
			fmt.Sprintf("<%s>", emoji.GlobalID),
			util.CreateMessageURL(i.GuildID, channelID, messageID),
		),
		messages.ColorYellow,
	))
	messages.SendChannelMessage(i.ChannelID, strings.Join(mentions, " "))
}

func (h *PlayerHandler) HandleAutocomplete(_ *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	for _, opt := range opts {
		if !opt.Focused {
			continue
		}

		switch opt.Name {
		case PlayerTagOptionName:
			autocompletePlayers(i, h.players, opt.StringValue())
		case MyPlayerTagOptionName:
			h.autocompleteMyPlayers(i, opt.StringValue())
		}
	}
}

func (h *PlayerHandler) autocompleteMyPlayers(i *discordgo.InteractionCreate, query string) {
	players, err := h.players.MyPlayers(i.Member.User.ID, query)
	if err != nil {
		messages.SendAutoCompletion(i, nil)
		return
	}

	messages.SendAutoCompletion(i, players.Choices())
}
