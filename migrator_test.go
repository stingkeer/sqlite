package sqlite

import (
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	dsn := "file:" + name + "?mode=memory&cache=shared"
	db, err := gorm.Open(&Dialector{DSN: dsn}, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	return db
}

func tableRootpage(t *testing.T, db *gorm.DB, table string) int {
	t.Helper()
	var rootpage int
	if err := db.Raw("SELECT rootpage FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&rootpage).Error; err != nil {
		t.Fatalf("read rootpage for %s: %v", table, err)
	}
	return rootpage
}

// noRebuildModel mixes the patterns that previously made GORM rebuild the table on every
// startup: a parenthesized default on a time column (read back without its opening
// parenthesis, so the default never matched), plus sized / not-null string columns whose size
// this driver does not round-trip.
type noRebuildModel struct {
	ID    uint      `gorm:"primaryKey"`
	Name  string    `gorm:"size:255;not null"`
	Note  string    `gorm:"size:100;default:'x'"`
	Count int
	At    time.Time `gorm:"default:(CURRENT_TIMESTAMP)"`
}

func (noRebuildModel) TableName() string { return "no_rebuild_models" }

// TestAutoMigrateNoRebuild ensures that re-running AutoMigrate on an already migrated table
// does not rebuild it. Before the MigrateColumn override this failed: the table was dropped and
// recreated on every run (rootpage changed), copying all rows each time.
func TestAutoMigrateNoRebuild(t *testing.T) {
	db := openTestDB(t, "test_no_rebuild")
	if err := db.Migrator().DropTable(&noRebuildModel{}); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	if err := db.AutoMigrate(&noRebuildModel{}); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.Create(&noRebuildModel{Name: "n", Count: 1, At: time.Now()}).Error; err != nil {
		t.Fatalf("create row: %v", err)
	}

	table := noRebuildModel{}.TableName()
	before := tableRootpage(t, db, table)

	for i := 0; i < 3; i++ {
		if err := db.AutoMigrate(&noRebuildModel{}); err != nil {
			t.Fatalf("re-migrate %d: %v", i, err)
		}
	}

	if after := tableRootpage(t, db, table); after != before {
		t.Fatalf("AutoMigrate rebuilt a migrated table: rootpage %d -> %d", before, after)
	}

	var count int64
	if err := db.Model(&noRebuildModel{}).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("data lost after re-migrate: got %d rows, want 1", count)
	}
}

type genuineV1 struct {
	ID    uint `gorm:"primaryKey"`
	Value int
}

func (genuineV1) TableName() string { return "genuine_models" }

type genuineV2 struct {
	ID    uint `gorm:"primaryKey"`
	Value string `gorm:"size:64"`
}

func (genuineV2) TableName() string { return "genuine_models" }

// TestAutoMigrateGenuineTypeChange ensures the override still migrates a real base-type change
// (int -> string), i.e. we did not disable legitimate migrations.
func TestAutoMigrateGenuineTypeChange(t *testing.T) {
	db := openTestDB(t, "test_genuine_change")
	if err := db.Migrator().DropTable(&genuineV1{}); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	if err := db.AutoMigrate(&genuineV1{}); err != nil {
		t.Fatalf("migrate v1: %v", err)
	}

	valueType := func() string {
		t.Helper()
		cts, err := db.Migrator().ColumnTypes(&genuineV1{})
		if err != nil {
			t.Fatalf("column types: %v", err)
		}
		for _, ct := range cts {
			if ct.Name() == "value" {
				return normalizeSqliteType(ct.DatabaseTypeName())
			}
		}
		t.Fatalf("column %q not found", "value")
		return ""
	}

	if got := valueType(); got != "integer" {
		t.Fatalf("initial value type = %q, want integer", got)
	}

	if err := db.AutoMigrate(&genuineV2{}); err != nil {
		t.Fatalf("migrate v2: %v", err)
	}

	if got := valueType(); got != "text" {
		t.Fatalf("value type after genuine change = %q, want text", got)
	}
}
