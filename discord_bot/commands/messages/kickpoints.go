package messages

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"

	"bot/commands/util"
	"bot/store/postgres/models"
	"bot/types"
)

func SendClanKickpoints(i *discordgo.InteractionCreate, clanName string, members []*types.ClanMemberKickpoints) {
	desc := "" // send in description because of embed field limit
	for _, m := range members {
		desc += fmt.Sprintf("%s (%s): %d Kickpunkte\n\n", m.Name, m.Tag, m.Amount)
	}

	SendEmbedResponse(i, NewEmbed(
		fmt.Sprintf("Kickpunkte von %s", clanName),
		desc,
		ColorAqua,
	))
}

func SendMemberKickpoints(i *discordgo.InteractionCreate, kickpoints []*models.Kickpoint, kickpointSum int) {
	fields := make([]*discordgo.MessageEmbedField, len(kickpoints)+1)
	var sum int
	for index, k := range kickpoints {
		sum += k.Amount
		field := &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("Kickpunkt #%d", k.ID),
		}

		for _, f := range DetailedKickpointFields(k) {
			field.Value += fmt.Sprintf("%s: %s\n", f.Name, f.Value)
		}

		fields[index] = field
	}

	field := discordgo.MessageEmbedField{
		Name:   "Gesamtanzahl (Vergangene und aktuelle Kickpunkte)",
		Value:  strconv.Itoa(kickpointSum),
		Inline: true,
	}
	fields[len(kickpoints)] = &field

	player := kickpoints[0].Player
	SendEmbedResponse(i, NewFieldEmbed(
		"Aktive Kickpunkte",
		fmt.Sprintf("Aktive Kickpunkte von %s (%s) in %s\n**Gesamt: %d/%d Kickpunkte**", player.Name, player.CocTag, "settings.Clan.Name", sum, 999),
		ColorAqua,
		fields,
	))
}

func DetailedKickpointFields(kickpoint *models.Kickpoint) []*discordgo.MessageEmbedField {
	return []*discordgo.MessageEmbedField{
		{Name: "Grund", Value: kickpoint.Description},
		{Name: "Anzahl Kickpunkte", Value: strconv.Itoa(kickpoint.Amount), Inline: true},
		{Name: "Erhalten am", Value: util.FormatDate(kickpoint.Date), Inline: true},
		{Name: "Läuft ab in", Value: util.FormatDuration(time.Until(kickpoint.ExpiresAt)), Inline: true},
		{Name: "Aktiv seit", Value: util.FormatDuration(time.Since(kickpoint.Date)), Inline: true},
		{Name: "Erstellt", Value: util.FormatFromAt(kickpoint.CreatedByUser, kickpoint.CreatedAt), Inline: true},
		{Name: "Zuletzt bearbeitet", Value: util.FormatFromAt(kickpoint.UpdatedByUser, kickpoint.UpdatedAt), Inline: true},
	}
}

func SendKickpointHelp(i *discordgo.InteractionCreate) {
	fields := []*discordgo.MessageEmbedField{
		{
			Name: "Keine Verwarnungen mehr",
			Value: `
Die größte Änderung im System des Bots ist, dass es keine Verwarnungen mehr gibt. Alle Regelbrüche, welche vorher Verwarnungen gegeben haben, geben jetzt auch Kickpunkte.
Für Regelbrüche, die im anderen System schon Kickpunkte gegeben haben, kriegt ein Mitglied jetzt mehr Kickpunkte. Natürlich wird damit auch die Anzahl erlaubter Kickpunkte deutlich erhöht.
Dies macht das System einfacher und übersichtlicher.
`,
		},
		{
			Name: "Fixe Ablaufdauer von Kickpunkten",
			Value: `
Leader von einem Clan können einstellen, wie lange Kickpunkte aktiv sind (bspw. 45 Tage). Nach dieser Zeit verfällt ein Kickpunkt automatisch, unabhängig davon, ob in dieser Zeit neue Punkte geholt wurden.
Somit können beim Entfernen und Hinzufügen von Kickpunkten keine Fehler mehr passieren.
`,
		},
		{
			Name: "Fairerer Abbau und fairere Verteilung",
			Value: `
Fairerer Abbau: Wenn man im alten System einen Kickpunkt am 01.01. und am 31.01. erhalten hat, wurden diese gleichzeitig abgebaut, obwohl 30 Tage dazwischen liegen.
Im neuen System wird das Abbaudatum genau X Tage nach dem Erhalt des Kickpunktes sein. (X = vom Leader eingestellte Anzahl an Tagen)

Fairere Verteilung: Im neuen System können Kickpunkte für "schlimmere" Regelbrüche mehr Punkte geben, als für "weniger schlimme" Regelbrüche.
Ein Beispiel dafür wäre ein verpasster CKL Angriff im Vergleich zu einem 0-Star, welche im alten System (je nach Clan) beide 1 Kickpunkt geben.
`,
		},
		{
			Name: "Befehl `kpmember [clan_tag] [player_tag]`",
			Value: `
Mit diesem Befehl kann jedes Mitglied einsehen
- Wie viele Kickpunkte es hat
- Für welche Regelbrüche es Kickpunkte erhalten hat
- Wann es die Kickpunkte erhalten hat
- Wann die Kickpunkte ablaufen
- Von wem es den Kickpunkt erhalten hat
`,
		},
		{
			Name: "Befehl `kpclan [clan_tag]`",
			Value: `
Mit diesem Befehl können Mitglieder einsehen, wie viele Kickpunkte jedes Mitglied im Clan hat.
`,
		},
		{
			Name: "Befehl `kpinfo [clan_tag]`",
			Value: `
Mit diesem Befehl kann jedes Mitglied einsehen
- Wie hoch die maximale Anzahl an Kickpunkten bis zum Kick ist
- Nach wie vielen Tagen Kickpunkte ablaufen
- Wie viele Kickpunkte für welchen Regelbruch vergeben werden
`,
		},
	}

	SendEmbedResponse(i, NewFieldEmbed(
		"Neues Kickpunktesystem - Erklärung",
		"Das Kickpunkte System vom Bot funktioniert anders, als das alte System. Hier sind die wichtigsten Änderungen:",
		ColorAqua,
		fields,
	))
}
