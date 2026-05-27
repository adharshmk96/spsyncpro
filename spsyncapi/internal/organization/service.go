package organization

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"spsyncapi/internal/crypto"
	"spsyncapi/internal/storage"

	"github.com/google/uuid"
)

var (
	ErrInvalidName         = errors.New("name is required")
	ErrInvalidTenantID     = errors.New("tenant_id is required")
	ErrInvalidClientID     = errors.New("client_id is required")
	ErrInvalidTenantSecret = errors.New("tenant_secret is required")
	ErrInvalidMemberID     = errors.New("member id is required")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrTenantIDTaken       = errors.New("tenant id already registered")
)

// OrganizationDetails is the public API representation (secret never included).
type OrganizationDetails struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TenantID  string    `json:"tenant_id"`
	ClientID  string    `json:"client_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateInput holds fields for creating an organization.
type CreateInput struct {
	MemberID     string
	Name         string
	TenantID     string
	ClientID     string
	TenantSecret string
}

// UpdateInput holds fields for updating an organization.
type UpdateInput struct {
	ID           string
	Name         string
	TenantID     string
	ClientID     string
	TenantSecret string // optional; empty means keep existing
}

// Service orchestrates organization business logic.
type Service struct {
	repo      *storage.OrganizationRepository
	encryptor *crypto.SecretEncryptor
	logger    *slog.Logger
}

// ServiceConfig wires dependencies for Service.
type ServiceConfig struct {
	Repo      *storage.OrganizationRepository
	Encryptor *crypto.SecretEncryptor
	Logger    *slog.Logger
}

// NewService constructs an organization Service.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Repo == nil {
		return nil, errors.New("organization repo is required")
	}
	if cfg.Encryptor == nil {
		return nil, errors.New("secret encryptor is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("logger is required")
	}
	return &Service{
		repo:      cfg.Repo,
		encryptor: cfg.Encryptor,
		logger:    cfg.Logger,
	}, nil
}

// Create registers a new organization with an encrypted tenant secret.
func (s *Service) Create(in CreateInput) (*OrganizationDetails, error) {
	if strings.TrimSpace(in.MemberID) == "" {
		return nil, ErrInvalidMemberID
	}
	if err := validateCreate(in); err != nil {
		return nil, err
	}

	encrypted, err := s.encryptor.Encrypt(strings.TrimSpace(in.TenantSecret))
	if err != nil {
		return nil, fmt.Errorf("encrypt tenant secret: %w", err)
	}

	now := time.Now().UTC()
	org := &storage.Organization{
		ID:                    uuid.NewString(),
		MemberID:              in.MemberID,
		Name:                  in.Name,
		TenantID:              in.TenantID,
		ClientID:              in.ClientID,
		TenantSecretEncrypted: encrypted,
		Active:                true,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.repo.Create(org); err != nil {
		if errors.Is(err, storage.ErrTenantIDTaken) {
			return nil, ErrTenantIDTaken
		}
		return nil, fmt.Errorf("create organization: %w", err)
	}

	return toDetails(org), nil
}

// Get returns an active organization by ID for the given member.
func (s *Service) Get(memberID, id string) (*OrganizationDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	org, err := s.repo.FindActiveByID(id, memberID)
	if err != nil {
		if errors.Is(err, storage.ErrOrganizationNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return toDetails(org), nil
}

// List returns all active organizations for the given member.
func (s *Service) List(memberID string) ([]OrganizationDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	orgs, err := s.repo.ListActive(memberID)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	out := make([]OrganizationDetails, 0, len(orgs))
	for i := range orgs {
		out = append(out, *toDetails(&orgs[i]))
	}
	return out, nil
}

// Update modifies an active organization. Tenant secret is re-encrypted when provided.
func (s *Service) Update(memberID string, in UpdateInput) (*OrganizationDetails, error) {
	if strings.TrimSpace(memberID) == "" {
		return nil, ErrInvalidMemberID
	}
	if strings.TrimSpace(in.ID) == "" {
		return nil, ErrOrganizationNotFound
	}
	if err := validateUpdate(in); err != nil {
		return nil, err
	}

	org, err := s.repo.FindActiveByID(in.ID, memberID)
	if err != nil {
		if errors.Is(err, storage.ErrOrganizationNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("find organization: %w", err)
	}

	if tenantID := strings.TrimSpace(in.TenantID); tenantID != "" && tenantID != org.TenantID {
		existing, findErr := s.repo.FindByTenantID(tenantID, memberID)
		if findErr == nil && existing.ID != org.ID {
			return nil, ErrTenantIDTaken
		}
		if findErr != nil && !errors.Is(findErr, storage.ErrOrganizationNotFound) {
			return nil, fmt.Errorf("check tenant id: %w", findErr)
		}
	}

	org.Name = in.Name
	org.TenantID = in.TenantID
	org.ClientID = in.ClientID
	org.UpdatedAt = time.Now().UTC()

	if secret := strings.TrimSpace(in.TenantSecret); secret != "" {
		encrypted, encErr := s.encryptor.Encrypt(secret)
		if encErr != nil {
			return nil, fmt.Errorf("encrypt tenant secret: %w", encErr)
		}
		org.TenantSecretEncrypted = encrypted
	}

	if err := s.repo.Update(org); err != nil {
		if errors.Is(err, storage.ErrTenantIDTaken) {
			return nil, ErrTenantIDTaken
		}
		return nil, fmt.Errorf("update organization: %w", err)
	}

	return toDetails(org), nil
}

// Delete marks an organization inactive (soft delete).
func (s *Service) Delete(memberID, id string) error {
	if strings.TrimSpace(memberID) == "" {
		return ErrInvalidMemberID
	}
	if err := s.repo.MarkInactive(id, memberID); err != nil {
		if errors.Is(err, storage.ErrOrganizationNotFound) {
			return ErrOrganizationNotFound
		}
		return fmt.Errorf("delete organization: %w", err)
	}
	return nil
}

func validateCreate(in CreateInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return ErrInvalidName
	}
	if strings.TrimSpace(in.TenantID) == "" {
		return ErrInvalidTenantID
	}
	if strings.TrimSpace(in.ClientID) == "" {
		return ErrInvalidClientID
	}
	if strings.TrimSpace(in.TenantSecret) == "" {
		return ErrInvalidTenantSecret
	}
	return nil
}

func validateUpdate(in UpdateInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return ErrInvalidName
	}
	if strings.TrimSpace(in.TenantID) == "" {
		return ErrInvalidTenantID
	}
	if strings.TrimSpace(in.ClientID) == "" {
		return ErrInvalidClientID
	}
	return nil
}

func toDetails(org *storage.Organization) *OrganizationDetails {
	return &OrganizationDetails{
		ID:        org.ID,
		Name:      org.Name,
		TenantID:  org.TenantID,
		ClientID:  org.ClientID,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}
}
