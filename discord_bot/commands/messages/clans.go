package messages

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

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
	name      string
	discordID string
	warPos    int
}

func getDonatorRanges(size int) [][][]int {
	ranges := map[int][][][]int{
		5:  {{{1, 3}, {4, 5}}, {{4, 5}, {1, 3}}}, // {{donator1}, {donator2}}
		10: {{{1, 5}, {6, 10}}, {{6, 10}, {1, 5}}},
		15: {{{1, 7}, {8, 15}}, {{8, 15}, {1, 7}}},
		20: {{{1, 10}, {11, 20}}, {{11, 20}, {1, 10}}},
		25: {{{1, 12}, {13, 25}}, {{13, 25}, {1, 12}}},
		30: {{{1, 10}, {11, 20}}, {{11, 20}, {21, 30}}, {{21, 30}, {1, 10}}},
		35: {{{1, 11}, {12, 35}}, {{12, 23}, {24, 35}}, {{24, 35}, {1, 11}}},
		40: {{{1, 10}, {11, 20}}, {{11, 20}, {1, 10}}, {{21, 30}, {31, 40}}, {{31, 40}, {21, 30}}},
		45: {{{1, 11}, {12, 22}}, {{12, 22}, {1, 11}}, {{23, 33}, {34, 45}}, {{34, 45}, {23, 33}}},
		50: {{{1, 10}, {11, 20}}, {{11, 20}, {1, 10}}, {{21, 30}, {31, 40}}, {{31, 40}, {41, 50}}, {{41, 50}, {21, 30}}},
	}
	return ranges[size]
}
func getRandom(min, max int) int {
	return rand.Intn(max-min+1) + min
}

func SendCWDonatorPing(s *discordgo.Session, i *discordgo.InteractionCreate, members []*models.ClanMember, ClanWarMembers []goclash.ClanWarMember, clanMemberByTag map[string]goclash.Player) {
	memberByTag := make(map[string]models.ClanMember, len(members))
	for _, member := range members {
		if member.Player == nil {
			continue
		}
		memberByTag[member.PlayerTag] = *member
	}

	var cwMembers []cwMember
	for _, clanWarMember := range ClanWarMembers {
		clanMember, clanMemberExists := clanMemberByTag[clanWarMember.Tag]
		if !clanMemberExists {
			println("Error: clanMember not found for tag", clanWarMember.Tag)
			continue
		}

		member, memberExists := memberByTag[clanMember.Tag]

		if member.Player == nil {
			println("Error: member.Player is nil for tag", clanMember.Tag)
			continue
		}

		if clanMember.WarPreference == "in" {
			if memberExists {
				cwMembers = append(cwMembers, cwMember{
					name:      clanWarMember.Name,
					discordID: member.Player.DiscordID,
					warPos:    clanWarMember.MapPosition,
				})
			} else {
				cwMembers = append(cwMembers, cwMember{
					name:      clanWarMember.Name,
					discordID: "",
					warPos:    clanWarMember.MapPosition,
				})
			}
		}
	}

	time.Sleep(5 * time.Second)

	cwSize := len(ClanWarMembers)
	ranges := getDonatorRanges(cwSize)

	type Donator struct {
		int      int
		tag      string
		name     string
		rangeStr string
	}

	donators := make(map[string]Donator)

	rand.Seed(time.Now().UnixNano())

	keys := map[int]string{
		0: "first",
		1: "second",
		2: "third",
		3: "fourth",
		4: "fifth",
	}

	for {
		valid := true

		for i, r := range ranges {
			donation_range := r[0]
			donator_random_range := r[1]
			key := keys[i]
			donators[key] = Donator{
				int:      getRandom(donator_random_range[0], donator_random_range[1]),
				tag:      "",
				name:     "",
				rangeStr: fmt.Sprintf("%d-%d", donation_range[0], donation_range[1]),
			}
		}

		for _, member := range cwMembers {
			for key, donator := range donators {
				if donator.int == member.warPos {
					donators[key] = Donator{
						int:      donator.int,
						tag:      member.discordID,
						name:     member.name,
						rangeStr: donator.rangeStr,
					}
				}
			}
		}

		donatorMap := make(map[string]bool)
		for _, donator := range donators {
			if donator.tag == "" {
				valid = false
				break
			}
			if _, exists := donatorMap[donator.tag]; exists {
				valid = false
				break
			}
		}

		if valid {
			break
		}
	}

	var content string
	keysList := []string{"first", "second", "third", "fourth", "fifth"}
	for _, key := range keysList {
		donator := donators[key]
		if donator.tag != "" {
			content += fmt.Sprintf("%s: %s (<@%s>) (%d)\n", donator.rangeStr, donator.name, donator.tag, donator.int)
		}
	}

	if content == "" {
		content = "Es sind keine Mitglieder im Krieg."
	}

	editEmbed := NewEmbed("CW Spender", "Folgende Mitglieder wurden zufällig als Spender ausgewählt:", ColorAqua)

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{editEmbed},
	})
	if err != nil {
		println("Error editing message:", err)
	}

	// send new message to the same channel with the ping message
	channelID := i.ChannelID
	_, err = s.ChannelMessageSend(channelID, content)
	if err != nil {
		println("Error sending message:", err)
	}
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
