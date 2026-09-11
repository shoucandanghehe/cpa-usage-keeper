package repository

import (
	"errors"
	"fmt"
	"time"

	"cpa-usage-keeper/internal/entities"

	"gorm.io/gorm"
)

var ErrBillingOverlap = errors.New("billing window overlaps another cycle for this account")

func ListBillingAccountIdentities(db *gorm.DB) ([]entities.UsageIdentity, error) {
	rows := []entities.UsageIdentity{}
	err := db.Where("auth_type = ? AND identity <> ''", entities.UsageIdentityAuthTypeAuthFile).
		Order("identity asc").Find(&rows).Error
	return rows, err
}

func FindBillingAccountIdentity(db *gorm.DB, authIndex string) (entities.UsageIdentity, error) {
	var row entities.UsageIdentity
	err := db.Where("auth_type = ? AND identity = ?", entities.UsageIdentityAuthTypeAuthFile, authIndex).Take(&row).Error
	return row, err
}

func ListBillingAPIKeys(db *gorm.DB) ([]entities.CPAAPIKey, error) {
	// Deleted keys remain valid historical participants; aliases never identify a member.
	rows := []entities.CPAAPIKey{}
	err := db.Select("id, api_key, display_key, key_alias").Order("id asc").Find(&rows).Error
	return rows, err
}

func ListBillingBills(db *gorm.DB, authIndex string) ([]entities.BillingBill, error) {
	rows := []entities.BillingBill{}
	err := db.Where("auth_index = ?", authIndex).Order("cycle desc").Find(&rows).Error
	return rows, err
}

func ListLatestBillingBills(db *gorm.DB) ([]entities.BillingBill, error) {
	rows := []entities.BillingBill{}
	err := db.Raw(`SELECT b.* FROM billing_bills b
		WHERE NOT EXISTS (SELECT 1 FROM billing_bills newer
			WHERE newer.auth_index = b.auth_index AND newer.cycle > b.cycle)
		ORDER BY b.auth_index`).Scan(&rows).Error
	return rows, err
}

// SaveBillingBillSnapshot requires the caller's transaction to include the usage
// read and calculation. Overlap validation and replacement share that snapshot.
func SaveBillingBillSnapshot(tx *gorm.DB, row *entities.BillingBill) error {
	var overlaps int64
	if err := tx.Model(&entities.BillingBill{}).
		Where("auth_index = ? AND cycle <> ? AND window_start < ? AND window_end > ?",
			row.AuthIndex, row.Cycle, row.WindowEnd, row.WindowStart).Count(&overlaps).Error; err != nil {
		return err
	}
	if overlaps != 0 {
		return ErrBillingOverlap
	}
	var existing entities.BillingBill
	err := tx.Where("auth_index = ? AND cycle = ?", row.AuthIndex, row.Cycle).Take(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil {
		row.ID = existing.ID
		return tx.Model(&existing).Select("window_start", "window_end", "snapshot_json").Updates(row).Error
	}
	return tx.Create(row).Error
}

func DeleteBillingBill(db *gorm.DB, id int64) error {
	result := db.Delete(&entities.BillingBill{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// StreamBillingUsage reads only one account and bounded calendar candidates.
// Stored timestamps have variable fractions/offsets, so SQL's lexical range is
// only a conservative index filter; Go performs the authoritative [start,end)
// instant comparison without SQLite's millisecond rounding. A one-day margin
// covers every RFC3339 offset, including DST and historical project TZ changes.
// The caller holds one transaction across both hot and archive reads. Archive
// IDs are copied from hot IDs; hot wins if an ID exists in both tables.
func StreamBillingUsage(tx *gorm.DB, authIndex string, start, end time.Time, visit func(entities.UsageEvent) error) error {
	lower := start.UTC().Add(-24 * time.Hour).Format("2006-01-02")
	upper := end.UTC().Add(24*time.Hour).Format("2006-01-02") + "z"
	if end.UTC().Add(24*time.Hour).Year() > 9999 {
		upper = ":" // sorts after every four-digit RFC3339 year
	}
	const columns = `id, api_group_key, model, model_alias, auth_index, timestamp,
		service_tier, response_service_tier, reasoning_effort, endpoint, executor_type,
		failed, input_tokens, output_tokens, reasoning_tokens, cache_read_tokens,
		cache_creation_tokens, total_tokens`
	for _, table := range []string{"usage_events", "usage_events_archive"} {
		query := tx.Table(table).Select(columns).
			Where("auth_index = ? AND timestamp >= ? AND timestamp < ?", authIndex, lower, upper).
			Order("timestamp asc, id asc")
		if table == "usage_events_archive" {
			query = query.Where("NOT EXISTS (SELECT 1 FROM usage_events hot WHERE hot.id = usage_events_archive.id)")
		}
		if err := streamBillingRows(tx, query, start, end, visit); err != nil {
			return err
		}
	}
	return nil
}

func streamBillingRows(tx, query *gorm.DB, start, end time.Time, visit func(entities.UsageEvent) error) error {
	rows, err := query.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := databaseContext(tx).Err(); err != nil {
			return err
		}
		var event entities.UsageEvent
		if err := tx.ScanRows(rows, &event); err != nil {
			return fmt.Errorf("read billing usage event: %w", err)
		}
		if event.Timestamp.Before(start) || !event.Timestamp.Before(end) {
			continue
		}
		if err := visit(event); err != nil {
			return err
		}
	}
	return rows.Err()
}
