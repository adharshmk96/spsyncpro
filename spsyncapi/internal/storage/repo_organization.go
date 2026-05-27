package storage

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ErrOrganizationNotFound is returned when an organization lookup yields no active result.
var ErrOrganizationNotFound = errors.New("organization not found")

// ErrTenantIDTaken is returned when tenant_id is already registered.
var ErrTenantIDTaken = errors.New("tenant id already registered")

// OrganizationRepository provides persistence operations for Organization records.
type OrganizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository constructs an OrganizationRepository backed by db.
func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create inserts a new Organization.
func (r *OrganizationRepository) Create(o *Organization) error {
	o.TenantID = normaliseID(o.TenantID)
	o.ClientID = strings.TrimSpace(o.ClientID)
	o.Name = strings.TrimSpace(o.Name)

	if err := r.db.Create(o).Error; err != nil {
		if isUniqueConstraintErr(err) {
			return ErrTenantIDTaken
		}
		return fmt.Errorf("organization repo: create: %w", err)
	}
	return nil
}

// FindActiveByID returns an active organization owned by memberID.
func (r *OrganizationRepository) FindActiveByID(id, memberID string) (*Organization, error) {
	var o Organization
	err := r.db.Where("id = ? AND active = ? AND member_id = ?", id, true, memberID).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOrganizationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("organization repo: find by id: %w", err)
	}
	return &o, nil
}

// ListActive returns active organizations owned by memberID ordered by name.
func (r *OrganizationRepository) ListActive(memberID string) ([]Organization, error) {
	var orgs []Organization
	err := r.db.Where("active = ? AND member_id = ?", true, memberID).Order("name ASC").Find(&orgs).Error
	if err != nil {
		return nil, fmt.Errorf("organization repo: list: %w", err)
	}
	return orgs, nil
}

// Update persists changes to an existing organization row.
func (r *OrganizationRepository) Update(o *Organization) error {
	o.TenantID = normaliseID(o.TenantID)
	o.ClientID = strings.TrimSpace(o.ClientID)
	o.Name = strings.TrimSpace(o.Name)

	result := r.db.Save(o)
	if result.Error != nil {
		if isUniqueConstraintErr(result.Error) {
			return ErrTenantIDTaken
		}
		return fmt.Errorf("organization repo: update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrOrganizationNotFound
	}
	return nil
}

// MarkInactive sets active=false for the organization owned by memberID (soft delete).
func (r *OrganizationRepository) MarkInactive(id, memberID string) error {
	result := r.db.Model(&Organization{}).
		Where("id = ? AND active = ? AND member_id = ?", id, true, memberID).
		Updates(map[string]interface{}{
			"active":     false,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("organization repo: mark inactive: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrOrganizationNotFound
	}
	return nil
}

// FindByTenantID returns an organization with the given tenant_id for memberID.
func (r *OrganizationRepository) FindByTenantID(tenantID, memberID string) (*Organization, error) {
	var o Organization
	err := r.db.Where("tenant_id = ? AND member_id = ?", normaliseID(tenantID), memberID).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOrganizationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("organization repo: find by tenant id: %w", err)
	}
	return &o, nil
}

func normaliseID(value string) string {
	return strings.TrimSpace(value)
}
