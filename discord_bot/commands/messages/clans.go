package messages

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"

	"github.com/aaantiii/goclash"
	"github.com/alexeyco/simpletable"
	"github.com/bwmarrin/discordgo"

	"bot/commands/util"
	"bot/store/postgres/models"
	"bot/types"
)

func PlayerLeaderboardTable(playerStats types.PlayerStatistics) string {
	sort.SliceStable(playerStats, func(i, j int) bool {
		return playerStats[i].Value > playerStats[j].Value
	})

	table := simpletable.New()
	table.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignCenter, Text: "#"},
			{Align: simpletable.AlignCenter, Text: "Name"},
			{Align: simpletable.AlignCenter, Text: "Wert"},
		},
	}

	for i, stat := range playerStats {
		if stat.Placement <= 0 {
			stat.Placement = i + 1
		}
		r := []*simpletable.Cell{
			{Align: simpletable.AlignLeft, Text: fmt.Sprintf("%d", stat.Placement)},
			{Align: simpletable.AlignLeft, Text: stat.Name},
			{Align: simpletable.AlignRight, Text: util.FormatNumber(stat.Value)},
		}
		table.Body.Cells = append(table.Body.Cells, r)
	}

	table.SetStyle(simpletable.StyleCompactLite)
	return "```\n" + table.String() + "\n```"
}

type cwMember struct {
	tag       string
	name      string
	discordID string
	warPos    int
}

type cwDonator struct {
	cwMember
	donationRange cwDonatorRange
}

type cwDonatorRange struct {
	start int
	end   int
}

func getDonatorRanges(cwSize int) []cwDonatorRange {
	numRanges := int(math.Ceil(float64(cwSize) / 10))
	rangeSize := int(math.Ceil(float64(cwSize) / float64(numRanges)))

	donatorRanges := make([]cwDonatorRange, numRanges)
	for i := 0; i < numRanges; i++ {
		start := i*rangeSize + 1
		end := start + rangeSize - 1
		if end > cwSize {
			end = cwSize
		}
		donatorRanges[i] = cwDonatorRange{start: start, end: end}
	}

	return donatorRanges
}

func CWDonatorPing(members []*models.ClanMember, clanWarMembers []goclash.ClanWarMember, clanMemberByTag map[string]*goclash.Player) string {
	memberByTag := make(map[string]*models.ClanMember, len(members))
	for _, member := range members {
		if member.Player == nil {
			continue
		}
		memberByTag[member.PlayerTag] = member
	}

	var possibleDonators []cwMember
	for _, clanWarMember := range clanWarMembers {
		clanMember, found := clanMemberByTag[clanWarMember.Tag]
		if !found {
			slog.Error("clanMember not found", slog.String("tag", clanWarMember.Tag))
			continue
		}

		if clanMember.WarPreference != "in" {
			continue
		}

		discordID := ""
		if member, memberExists := memberByTag[clanMember.Tag]; memberExists && member.Player != nil {
			discordID = member.Player.DiscordID
		}

		possibleDonators = append(possibleDonators, cwMember{
			tag:       clanWarMember.Tag,
			name:      clanWarMember.Name,
			discordID: discordID,
			warPos:    clanWarMember.MapPosition,
		})
	}

	if len(possibleDonators) == 0 {
		return "Es sind keine Mitglieder im Krieg."
	}

	cwSize := len(clanWarMembers)
	ranges := getDonatorRanges(cwSize)
	donators := make([]cwDonator, len(ranges))

	uniqueRand := util.NewUniqueRand()
	for index := range donators {
		randomDonator := possibleDonators[uniqueRand.Intn(0, len(possibleDonators)-1)]
		donators[index] = cwDonator{
			cwMember:      randomDonator,
			donationRange: ranges[index],
		}
	}

	content := ""
	for _, donator := range donators {
		content += fmt.Sprintf("%d-%d: %s", donator.donationRange.start, donator.donationRange.end, donator.name)
		if donator.discordID != "" {
			content += fmt.Sprintf(" (<@%s>)", donator.discordID)
		}
		content += fmt.Sprintf(" (Nr. %d)\n", donator.warPos)
	}

	return content
}

type raidPingMember struct {
	name         string
	discordID    string
	attacks      int
	totalAttacks int
}

func SendRaidPing(i *discordgo.InteractionCreate, members models.ClanMembers, raidSeason goclash.ClanCapitalRaidSeason) {
	raidMemberByTag := make(map[string]goclash.ClanCapitalRaidSeasonMember, len(raidSeason.Members))
	for _, m := range raidSeason.Members {
		raidMemberByTag[m.Tag] = m
	}

	var completelyMissing []raidPingMember
	var attacksMissing []raidPingMember
	for _, member := range members {
		raidMember, ok := raidMemberByTag[member.PlayerTag]
		if !ok {
			completelyMissing = append(completelyMissing, raidPingMember{
				name:      member.Player.Name,
				discordID: member.Player.DiscordID,
			})
			continue
		}
		if raidMember.Attacks < raidMember.AttackLimit+raidMember.BonusAttackLimit {
			attacksMissing = append(attacksMissing, raidPingMember{
				name:         member.Player.Name,
				discordID:    member.Player.DiscordID,
				attacks:      raidMember.Attacks,
				totalAttacks: raidMember.AttackLimit + raidMember.BonusAttackLimit,
			})
		}
	}

	var content string
	if len(completelyMissing) > 0 {
		content += "**Noch gar nicht angegriffen:**\n"
		for _, member := range completelyMissing {
			content += fmt.Sprintf("%s\n", util.MentionUserID(member.discordID))
		}
	}

	if len(attacksMissing) > 0 {
		content += "\n**Noch offene Angriffe:**\n"
		for _, member := range attacksMissing {
			content += fmt.Sprintf("%s (%d/%d)\n", util.MentionUserID(member.discordID), member.attacks, member.totalAttacks)
		}
	}

	if content == "" {
		SendEmbedResponse(i, NewEmbed("Alle Angriffe erledigt", "Es sind keine Angriffe mehr offen!", ColorGreen))
		return
	}

	SendMessageResponse(i, "Fehlende Raid Angriffe", content)
}

func EventEmbedFields(event *models.ClanEvent, playerStats types.PlayerStatistics) []*discordgo.MessageEmbedField {
	compStat := util.ComparableStatisticByName(event.StatName)
	fields := []*discordgo.MessageEmbedField{
		{Name: "ID", Value: strconv.Itoa(int(event.ID)), Inline: true},
		{Name: "Clan", Value: event.Clan.Name, Inline: true},
		{Name: "Aufgabe", Value: compStat.Task, Inline: true},
		{Name: "Start", Value: util.FormatDateTime(event.StartsAt), Inline: true},
		{Name: "Ende", Value: util.FormatDateTime(event.EndsAt), Inline: true},
	}

	if event.WinnerPlayerTag != nil && playerStats != nil {
		for _, player := range playerStats {
			if player.Tag == *event.WinnerPlayerTag {
				fields = append(fields, &discordgo.MessageEmbedField{Name: "Gewinner", Value: player.Name, Inline: true})
				break
			}
		}
	}

	return fields
}
