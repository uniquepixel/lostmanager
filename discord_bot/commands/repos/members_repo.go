package repos

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"bot/store/postgres/models"
)

type IMembersRepo interface {
	MembersByClanTag(clanTag string) (models.ClanMembers, error)
	MembersByDiscordID(discordID string) (models.ClanMembers, error)
	MemberByID(playerTag, clanTag string) (*models.ClanMember, error)
	MembersByTag(clanTag string, playerTags ...string) (models.ClanMembers, error)
	MembersByPlayerTag(playerTag string) (models.ClanMembers, error)
	GetPlayerCurrentClan(playerTag string) (*models.ClanMember, error)
	CreateMember(member *models.ClanMember) error
	TransferMember(playerTag, fromClanTag, toClanTag string, newRole models.ClanRole, transferredByDiscordID string) error
	UpdateMemberRole(playerTag, clanTag string, role models.ClanRole) error
	DeleteMember(tag, clanTag string) error
}

type MembersRepo struct {
	db *gorm.DB
}

func NewMembersRepo(db *gorm.DB) IMembersRepo {
	return &MembersRepo{db: db}
}

func (repo *MembersRepo) MembersByClanTag(clanTag string) (models.ClanMembers, error) {
	var members models.ClanMembers
	if err := repo.db.
		Preload("Player").
		Find(&members, "clan_tag = ?", clanTag).Error; err != nil {
		return nil, err
	}

	return members, nil
}

func (repo *MembersRepo) MembersByDiscordID(discordID string) (models.ClanMembers, error) {
	var players []*models.Player
	err := repo.db.
		Preload("ClanMembers").
		Find(&players, "discord_id = ?", discordID).Error

	var members models.ClanMembers
	for _, player := range players {
		members = append(members, player.Members...)
	}

	return members, err
}

func (repo *MembersRepo) MemberByID(playerTag, clanTag string) (*models.ClanMember, error) {
	var members *models.ClanMember
	err := repo.db.
		Preload(clause.Associations).
		First(&members, "player_tag = ? AND clan_tag = ?", playerTag, clanTag).Error
	return members, err
}

func (repo *MembersRepo) MembersByTag(clanTag string, playerTags ...string) (models.ClanMembers, error) {
	var members models.ClanMembers
	err := repo.db.
		Preload(clause.Associations).
		Find(&members, "clan_tag = ? AND player_tag IN (?)", clanTag, playerTags).Error
	return members, err
}

func (repo *MembersRepo) MembersByPlayerTag(playerTag string) (models.ClanMembers, error) {
	var members models.ClanMembers
	err := repo.db.
		Preload(clause.Associations).
		Find(&members, "player_tag = ?", playerTag).Error
	return members, err
}

func (repo *MembersRepo) GetPlayerCurrentClan(playerTag string) (*models.ClanMember, error) {
	var member *models.ClanMember
	err := repo.db.
		Preload(clause.Associations).
		Order("joined_at DESC"). // Ensure the most recent clan is selected
		First(&member, "player_tag = ?", playerTag).Error
	return member, err
}

func (repo *MembersRepo) MissingClanMembers(clanTag string, playerTags ...string) (models.ClanMembers, error) {
	var members models.ClanMembers
	err := repo.db.
		Preload(clause.Associations).
		Find(&members, "clan_tag = ? AND player_tag NOT IN (?)", clanTag, playerTags).Error
	return members, err
}

func (repo *MembersRepo) CreateMember(member *models.ClanMember) error {
	return repo.db.Create(member).Error
}

func (repo *MembersRepo) TransferMember(playerTag, fromClanTag, toClanTag string, newRole models.ClanRole, transferredByDiscordID string) error {
	return repo.db.Transaction(func(tx *gorm.DB) error {
		// Delete from old clan
		if err := tx.Delete(&models.ClanMember{}, "player_tag = ? AND clan_tag = ?", playerTag, fromClanTag).Error; err != nil {
			return err
		}
		
		// Add to new clan
		newMember := &models.ClanMember{
			PlayerTag:        playerTag,
			ClanTag:          toClanTag,
			ClanRole:         newRole,
			AddedByDiscordID: transferredByDiscordID,
		}
		
		return tx.Create(newMember).Error
	})
}

func (repo *MembersRepo) UpdateMemberRole(playerTag, clanTag string, role models.ClanRole) error {
	return repo.db.
		Model(&models.ClanMember{PlayerTag: playerTag, ClanTag: clanTag}).
		Update("clan_role", role).Error
}

func (repo *MembersRepo) DeleteMember(tag, clanTag string) error {
	return repo.db.Delete(&models.ClanMember{}, "player_tag = ? AND clan_tag = ?", tag, clanTag).Error
}
