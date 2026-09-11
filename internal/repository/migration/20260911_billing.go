package migration

import (
	"fmt"

	"cpa-usage-keeper/internal/entities"

	"gorm.io/gorm"
)

func createBillingMigration(db *gorm.DB) error {
	if err := db.AutoMigrate(&entities.BillingBill{}); err != nil {
		return fmt.Errorf("create billing snapshots: %w", err)
	}
	// The cold table has no unrelated online-query indexes. Billing needs only
	// this account/window range index; do not rebuild or duplicate its data.
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_usage_archive_auth_index_timestamp ON usage_events_archive (auth_index, timestamp)").Error; err != nil {
		return fmt.Errorf("index archived billing usage: %w", err)
	}
	return nil
}
