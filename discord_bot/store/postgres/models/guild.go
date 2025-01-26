package models

import (
	"slices"
)

const LostFamilyGuildID = "733857906117574717"

type Guild struct {
	GuildID        string `gorm:"primaryKey"`
	ClanTag        string `gorm:"primaryKey"`
	LeaderRoleID   string
	CoLeaderRoleID string
	ElderRoleID    string
	MemberRoleID   string
}

type ServerRoles struct {
	LeaderRoleIDs   []string
	CoLeaderRoleIDs []string
	ElderRoleIDs    []string
	MemberRoleIDs   []string
}

func (g *Guild) IsLeader(roles []string) bool {
	return slices.Contains(roles, g.LeaderRoleID)
}

func (g *Guild) IsCoLeader(roles []string) bool {
	return slices.Contains(roles, g.CoLeaderRoleID)
}

func (g *Guild) IsElder(roles []string) bool {
	return slices.Contains(roles, g.ElderRoleID)
}

func (g *Guild) IsMember(roles []string) bool {
	return slices.Contains(roles, g.MemberRoleID)
}

func (g *Guild) RoleIDByClanRole(cocRole ClanRole) string {
	switch cocRole {
	case RoleLeader:
		return g.LeaderRoleID
	case RoleCoLeader:
		return g.CoLeaderRoleID
	case RoleElder:
		return g.ElderRoleID
	case RoleMember:
		return g.MemberRoleID
	default:
		return ""
	}
}
