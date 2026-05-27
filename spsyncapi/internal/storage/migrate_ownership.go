package storage

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// migrateResourceOwnership backfills member_id on existing rows and replaces
// global unique indexes with per-member composite indexes.
func migrateResourceOwnership(db *gorm.DB) error {
	if err := backfillMemberOwnership(db); err != nil {
		return fmt.Errorf("backfill member ownership: %w", err)
	}
	if err := migrateOwnershipIndexes(db); err != nil {
		return fmt.Errorf("migrate ownership indexes: %w", err)
	}
	return nil
}

func backfillMemberOwnership(db *gorm.DB) error {
	var firstMember Member
	err := db.Order("created_at ASC").First(&firstMember).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find first member: %w", err)
	}

	tables := []string{"organizations", "bucket_stores", "backup_jobs"}
	for _, table := range tables {
		result := db.Exec(
			"UPDATE "+table+" SET member_id = ? WHERE member_id IS NULL OR member_id = ''",
			firstMember.ID,
		)
		if result.Error != nil {
			return fmt.Errorf("backfill %s: %w", table, result.Error)
		}
	}
	return nil
}

func migrateOwnershipIndexes(db *gorm.DB) error {
	legacyIndexes := []string{
		"idx_organizations_tenant_id",
		"organizations_tenant_id",
		"idx_bucket_stores_bucket_name",
		"bucket_stores_bucket_name",
	}
	for _, idx := range legacyIndexes {
		if err := db.Exec("DROP INDEX IF EXISTS " + idx).Error; err != nil {
			return fmt.Errorf("drop index %s: %w", idx, err)
		}
	}

	if err := db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_org_member_tenant ON organizations (member_id, tenant_id)",
	).Error; err != nil {
		return fmt.Errorf("create idx_org_member_tenant: %w", err)
	}

	if err := db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_bucket_member_name ON bucket_stores (member_id, bucket_name)",
	).Error; err != nil {
		return fmt.Errorf("create idx_bucket_member_name: %w", err)
	}

	return nil
}
