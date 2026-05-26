package storage

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ErrBucketStoreNotFound is returned when a bucket store lookup yields no active result.
var ErrBucketStoreNotFound = errors.New("bucket store not found")

// ErrBucketNameTaken is returned when bucket_name is already registered.
var ErrBucketNameTaken = errors.New("bucket name already registered")

// BucketStoreRepository provides persistence operations for BucketStore records.
type BucketStoreRepository struct {
	db *gorm.DB
}

// NewBucketStoreRepository constructs a BucketStoreRepository backed by db.
func NewBucketStoreRepository(db *gorm.DB) *BucketStoreRepository {
	return &BucketStoreRepository{db: db}
}

// Create inserts a new BucketStore.
func (r *BucketStoreRepository) Create(b *BucketStore) error {
	b.BucketName = normaliseBucketName(b.BucketName)
	b.BucketType = strings.TrimSpace(b.BucketType)

	if err := r.db.Create(b).Error; err != nil {
		if isUniqueConstraintErr(err) {
			return ErrBucketNameTaken
		}
		return fmt.Errorf("bucket store repo: create: %w", err)
	}
	return nil
}

// FindActiveByID returns an active bucket store by primary key.
func (r *BucketStoreRepository) FindActiveByID(id string) (*BucketStore, error) {
	var b BucketStore
	err := r.db.Where("id = ? AND active = ?", id, true).First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBucketStoreNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("bucket store repo: find by id: %w", err)
	}
	return &b, nil
}

// ListActive returns all active bucket stores ordered by bucket name.
func (r *BucketStoreRepository) ListActive() ([]BucketStore, error) {
	var stores []BucketStore
	err := r.db.Where("active = ?", true).Order("bucket_name ASC").Find(&stores).Error
	if err != nil {
		return nil, fmt.Errorf("bucket store repo: list: %w", err)
	}
	return stores, nil
}

// Update persists changes to an existing bucket store row.
func (r *BucketStoreRepository) Update(b *BucketStore) error {
	b.BucketName = normaliseBucketName(b.BucketName)
	b.BucketType = strings.TrimSpace(b.BucketType)

	result := r.db.Save(b)
	if result.Error != nil {
		if isUniqueConstraintErr(result.Error) {
			return ErrBucketNameTaken
		}
		return fmt.Errorf("bucket store repo: update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrBucketStoreNotFound
	}
	return nil
}

// MarkInactive sets active=false for the bucket store (soft delete).
func (r *BucketStoreRepository) MarkInactive(id string) error {
	result := r.db.Model(&BucketStore{}).
		Where("id = ? AND active = ?", id, true).
		Updates(map[string]interface{}{
			"active":     false,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("bucket store repo: mark inactive: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrBucketStoreNotFound
	}
	return nil
}

// FindByBucketName returns a bucket store with the given bucket_name (any active state).
func (r *BucketStoreRepository) FindByBucketName(name string) (*BucketStore, error) {
	var b BucketStore
	err := r.db.Where("bucket_name = ?", normaliseBucketName(name)).First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBucketStoreNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("bucket store repo: find by bucket name: %w", err)
	}
	return &b, nil
}

func normaliseBucketName(value string) string {
	return strings.TrimSpace(value)
}
