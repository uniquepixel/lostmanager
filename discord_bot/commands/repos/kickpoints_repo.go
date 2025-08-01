package repos

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"bot/commands/util"
	"bot/store/postgres/models"
	"bot/types"
)

type IKickpointsRepo interface {
	KickpointByID(id uint) (*models.Kickpoint, error)
	ActiveClanKickpoints(settings *models.ClanSettings) ([]*types.ClanMemberKickpoints, error)
	ActiveMemberKickpoints(memberTag string) ([]*models.Kickpoint, error)
	ActiveMemberKickpointsSum(memberTag string) (int, error)
	FutureMemberKickpoints(memberTag string) ([]*models.Kickpoint, error)
	KickpointSum(memberTag string) (int, error)
	CreateKickpoint(kickpoint *models.Kickpoint) error
	UpdateKickpoint(kickpoint *models.Kickpoint) (*models.Kickpoint, error)
	DeleteKickpoint(id uint) error
}

type KickpointsRepo struct {
	db *gorm.DB
}

func NewKickpointsRepo(db *gorm.DB) IKickpointsRepo {
	return &KickpointsRepo{db: db}
}

func (repo *KickpointsRepo) KickpointByID(id uint) (*models.Kickpoint, error) {
	var kickpoint *models.Kickpoint
	err := repo.db.Preload(clause.Associations).First(&kickpoint, id).Error
	return kickpoint, err
}

func (repo *KickpointsRepo) ActiveClanKickpoints(settings *models.ClanSettings) ([]*types.ClanMemberKickpoints, error) {
	minDate := util.KickpointMinDate(settings.KickpointsExpireAfterDays)

	var memberKickpoints []*types.ClanMemberKickpoints
	if err := repo.db.
		Raw("SELECT p.name AS name, p.coc_tag as tag, SUM(k.amount) AS amount FROM kickpoints k INNER JOIN players p ON k.player_tag = p.coc_tag INNER JOIN clan_members m ON p.coc_tag = m.player_tag WHERE m.clan_tag = ? AND k.date BETWEEN ? AND NOW() GROUP BY p.name, p.coc_tag ORDER BY amount DESC", settings.ClanTag, minDate).
		Scan(&memberKickpoints).Error; err != nil {
		return nil, err
	}

	if len(memberKickpoints) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return memberKickpoints, nil
}

func (repo *KickpointsRepo) ActiveMemberKickpoints(memberTag string) ([]*models.Kickpoint, error) {
	var kickpoints []*models.Kickpoint
	if err := repo.db.
		Preload(clause.Associations).
		Order("created_at").
		Find(&kickpoints, "player_tag = ? AND expires_at > NOW()", memberTag).Error; err != nil {
		return nil, err
	}

	if len(kickpoints) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return kickpoints, nil
}

func (repo *KickpointsRepo) ActiveMemberKickpointsSum(memberTag string) (int, error) {
	var v struct{ Sum int }
	if err := repo.db.
		Model(&models.Kickpoint{}).
		Where("player_tag = ? AND expires_at > NOW()", memberTag).
		Select("SUM(amount) as sum").
		Scan(&v).Error; err != nil {
		return 0, err
	}

	return v.Sum, nil
}

func (repo *KickpointsRepo) FutureMemberKickpoints(memberTag string) ([]*models.Kickpoint, error) {
	var kickpoints []*models.Kickpoint
	if err := repo.db.
		Preload(clause.Associations).
		Order("created_at").
		Limit(20).
		Find(&kickpoints, "player_tag = ? AND date > NOW()", memberTag).Error; err != nil {
		return nil, err
	}

	if len(kickpoints) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return kickpoints, nil
}

func (repo *KickpointsRepo) KickpointSum(memberTag string) (int, error) {
	var v struct{ Sum int }
	if err := repo.db.
		Model(&models.Kickpoint{}).
		Where("player_tag = ?", memberTag).
		Select("SUM(amount) as sum").
		Scan(&v).Error; err != nil {
		return 0, err
	}

	return v.Sum, nil
}

func (repo *KickpointsRepo) CreateKickpoint(kickpoint *models.Kickpoint) error {
	return repo.db.Create(&kickpoint).Error
}

func (repo *KickpointsRepo) UpdateKickpoint(kickpoint *models.Kickpoint) (*models.Kickpoint, error) {
	if err := repo.db.Updates(kickpoint).Error; err != nil {
		return nil, err
	}

	return repo.KickpointByID(kickpoint.ID)
}

func (repo *KickpointsRepo) DeleteKickpoint(id uint) error {
	return repo.db.Delete(&models.Kickpoint{}, id).Error
}
