package models

type MemberState struct {
	PlayerTag     string `gorm:"primaryKey"`
	ClanTag       string `gorm:"primaryKey"`
	KickpointLock bool   `gorm:"default:false;not null"`

	// Remove the association to prevent GORM from creating reverse foreign keys
	// Relationships will be handled manually in queries when needed
}
