package entities

// BillingBill keeps queryable identity/window metadata separate from its immutable
// calculation payload. SnapshotJSON never contains raw participant keys.
type BillingBill struct {
	ID           int64  `gorm:"primaryKey"`
	AuthIndex    string `gorm:"not null;uniqueIndex:uniq_billing_bills_account_cycle,priority:1;index:idx_billing_bills_account_window,priority:1"`
	Cycle        int64  `gorm:"not null;uniqueIndex:uniq_billing_bills_account_cycle,priority:2"`
	WindowStart  string `gorm:"not null;index:idx_billing_bills_account_window,priority:2"`
	WindowEnd    string `gorm:"not null"`
	SnapshotJSON string `gorm:"not null"`
}
