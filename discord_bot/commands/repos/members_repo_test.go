package repos

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"bot/store/postgres/models"
)

func TestCreateMember_PreventsDuplicatePlayers(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(&models.Player{}, &models.Clan{}, &models.ClanMember{}); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Create test data
	player := &models.Player{
		CocTag:    "#TESTPLAYER",
		Name:      "Test Player",
		DiscordID: "123456789",
	}
	clan1 := &models.Clan{
		Tag:  "#CLAN1",
		Name: "Test Clan 1",
	}
	clan2 := &models.Clan{
		Tag:  "#CLAN2",
		Name: "Test Clan 2",
	}

	if err := db.Create(player).Error; err != nil {
		t.Fatalf("Failed to create test player: %v", err)
	}
	if err := db.Create(clan1).Error; err != nil {
		t.Fatalf("Failed to create test clan 1: %v", err)
	}
	if err := db.Create(clan2).Error; err != nil {
		t.Fatalf("Failed to create test clan 2: %v", err)
	}

	repo := NewMembersRepo(db)

	// Test: Add player to first clan - should succeed
	member1 := &models.ClanMember{
		PlayerTag:        player.CocTag,
		ClanTag:          clan1.Tag,
		ClanRole:         models.RoleMember,
		AddedByDiscordID: "admin1",
	}

	if err := repo.CreateMember(member1); err != nil {
		t.Fatalf("Expected first member creation to succeed, got error: %v", err)
	}

	// Test: Try to add same player to second clan - should fail
	member2 := &models.ClanMember{
		PlayerTag:        player.CocTag,
		ClanTag:          clan2.Tag,
		ClanRole:         models.RoleMember,
		AddedByDiscordID: "admin2",
	}

	if err := repo.CreateMember(member2); err == nil {
		t.Fatal("Expected second member creation to fail, but it succeeded")
	}

	// Verify the error message contains expected text
	if err := repo.CreateMember(member2); err != nil {
		expectedText := "is already a member of clan"
		if !contains(err.Error(), expectedText) {
			t.Fatalf("Expected error to contain '%s', got: %v", expectedText, err)
		}
	}

	// Test: Verify player is only in first clan
	currentMember, err := repo.GetPlayerCurrentClan(player.CocTag)
	if err != nil {
		t.Fatalf("Failed to get current clan: %v", err)
	}

	if currentMember.ClanTag != clan1.Tag {
		t.Fatalf("Expected player to be in clan %s, but found in clan %s", clan1.Tag, currentMember.ClanTag)
	}
}

func TestTransferMember(t *testing.T) {
	// Setup in-memory database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(&models.Player{}, &models.Clan{}, &models.ClanMember{}); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Create test data
	player := &models.Player{
		CocTag:    "#TESTPLAYER",
		Name:      "Test Player",
		DiscordID: "123456789",
	}
	clan1 := &models.Clan{Tag: "#CLAN1", Name: "Test Clan 1"}
	clan2 := &models.Clan{Tag: "#CLAN2", Name: "Test Clan 2"}

	db.Create(player)
	db.Create(clan1)
	db.Create(clan2)

	repo := NewMembersRepo(db)

	// Add player to first clan
	member1 := &models.ClanMember{
		PlayerTag:        player.CocTag,
		ClanTag:          clan1.Tag,
		ClanRole:         models.RoleMember,
		AddedByDiscordID: "admin1",
	}
	if err := repo.CreateMember(member1); err != nil {
		t.Fatalf("Failed to create initial member: %v", err)
	}

	// Transfer player to second clan
	if err := repo.TransferMember(player.CocTag, clan1.Tag, clan2.Tag, models.RoleElder, "admin2"); err != nil {
		t.Fatalf("Failed to transfer member: %v", err)
	}

	// Verify player is now in second clan with new role
	currentMember, err := repo.GetPlayerCurrentClan(player.CocTag)
	if err != nil {
		t.Fatalf("Failed to get current clan after transfer: %v", err)
	}

	if currentMember.ClanTag != clan2.Tag {
		t.Fatalf("Expected player to be in clan %s after transfer, but found in clan %s", clan2.Tag, currentMember.ClanTag)
	}

	if currentMember.ClanRole != models.RoleElder {
		t.Fatalf("Expected player to have role %s after transfer, but has role %s", models.RoleElder, currentMember.ClanRole)
	}

	// Verify player is no longer in first clan
	_, err = repo.MemberByID(player.CocTag, clan1.Tag)
	if err == nil {
		t.Fatal("Expected player to not be in first clan after transfer, but member still exists")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || contains(s[1:len(s)-1], substr))))
}
