package repos

import (
	"sort"
	"strings"

	"gorm.io/gorm"

	"bot/store/postgres"
	"bot/store/postgres/models"
	"bot/types"
)

type IPlayersRepo interface {
	Players(query string) (models.Players, error)
	PlayersByDiscordID(discordID string, preload ...string) (models.Players, error)
	PlayerByTag(tag string) (*models.Player, error)
	PlayerByTagAndDiscordID(tag, discordID string) (*models.Player, error)
	CreateOrUpdatePlayer(player *models.Player) error
	NameByTag(tag string) (string, error)
	MembersPlayersByClan(clanTag, query string) (models.Players, error)
	MyPlayers(discordID string, query string) (models.Players, error)
}

type PlayersRepo struct {
	db *gorm.DB
}

func NewPlayersRepo(db *gorm.DB) IPlayersRepo {
	return &PlayersRepo{db: db}
}

func (repo *PlayersRepo) Players(query string) (models.Players, error) {
	var players models.Players
	err := repo.db.
		Preload("Member.Clan").
		Scopes(
			postgres.WithLimit(types.MaxCommandChoices),
			postgres.WithSearchQuery(query, "name", "coc_tag", "discord_id"),
		).
		Find(&players).Error
	return players, err
}

func (repo *PlayersRepo) PlayersByDiscordID(discordID string, preload ...string) (models.Players, error) {
	var players models.Players
	err := repo.db.
		Scopes(postgres.WithPreloading(preload...)).
		Find(&players, "discord_id = ?", discordID).Error
	return players, err
}

func (repo *PlayersRepo) PlayerByTag(tag string) (*models.Player, error) {
	var clan *models.Player
	err := repo.db.First(&clan, "coc_tag = ?", tag).Error
	return clan, err
}

func (repo *PlayersRepo) PlayerByTagAndDiscordID(tag, discordID string) (*models.Player, error) {
	var clan *models.Player
	err := repo.db.First(&clan, "coc_tag = ? AND discord_id = ?", tag, discordID).Error
	return clan, err
}

// CreateOrUpdatePlayer returns types.ErrNoChanges if player tag exists and discord id did not change.
func (repo *PlayersRepo) CreateOrUpdatePlayer(player *models.Player) error {
	return repo.db.Save(player).Error
}

func (repo *PlayersRepo) NameByTag(tag string) (string, error) {
	var name string
	err := repo.db.
		Model(&models.Player{}).
		Select("name").
		First(&name, "coc_tag = ?", tag).Error
	return name, err
}

func (repo *PlayersRepo) MembersPlayersByClan(clanTag, query string) (models.Players, error) {
	var players models.Players
	if err := repo.db.
		Scopes(postgres.WithSearchQuery(query, "coc_tag", "name")).
		Where("coc_tag IN (?)", repo.db.
			Model(&models.ClanMember{}).
			Select("player_tag").
			Where("clan_tag = ?", clanTag),
		).
		Limit(types.MaxCommandChoices).
		Find(&players).Error; err != nil {
		return nil, err
	}

	sort.Slice(players, func(i, _ int) bool {
		return strings.HasPrefix(strings.ToLower(players[i].Name), query)
	})

	return players, nil
}

func (repo *PlayersRepo) MyPlayers(discordID string, query string) (models.Players, error) {
	var players models.Players
	err := repo.db.
		Scopes(
			postgres.WithLimit(types.MaxCommandChoices),
			postgres.WithSearchQuery(query, "name", "coc_tag"),
		).
		Where("discord_id = ?", discordID).
		Find(&players).Error
	return players, err
}
