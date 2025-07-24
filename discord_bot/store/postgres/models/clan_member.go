package models

type ClanMember struct {
	PlayerTag        string `gorm:"primaryKey;not null"`
	ClanTag          string `gorm:"primaryKey;not null"`
	AddedByDiscordID string
	ClanRole         ClanRole

	Player *Player `gorm:"foreignKey:PlayerTag;references:CocTag;constraint:OnDelete:CASCADE"`
	Clan   *Clan   `gorm:"foreignKey:ClanTag;references:Tag;constraint:OnDelete:CASCADE"`
}

type ClanRole string

const (
	RoleLeader   ClanRole = "leader"
	RoleCoLeader ClanRole = "coLeader"
	RoleElder    ClanRole = "admin"
	RoleMember   ClanRole = "member"
)

func (r ClanRole) String() string {
	return string(r)
}

func (r ClanRole) Format() string {
	switch r {
	case RoleLeader:
		return "Anführer"
	case RoleCoLeader:
		return "Vize-Anführer"
	case RoleElder:
		return "Ältester"
	case RoleMember:
		return "Mitglied"
	default:
		return "Unbekannte Rolle"
	}
}

type ClanMembers []*ClanMember

func (members ClanMembers) Tags() []string {
	if members == nil {
		return nil
	}

	tags := make([]string, len(members))
	for i, member := range members {
		tags[i] = member.PlayerTag
	}

	return tags
}

func (members ClanMembers) TagsDistinct() []string {
	if len(members) == 0 {
		return nil
	}

	tags := make([]string, 1)
	seen := make(map[string]bool)
	for _, member := range members {
		if seen[member.PlayerTag] {
			continue
		}

		tags = append(tags, member.PlayerTag)
		seen[member.PlayerTag] = true
	}

	return tags
}
