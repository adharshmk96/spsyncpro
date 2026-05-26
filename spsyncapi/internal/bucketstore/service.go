package bucketstore

import (
	"encoding/json"
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
	ErrInvalidBucketName = errors.New("bucket_name is required")
	ErrInvalidBucketType = errors.New("bucket_type must be s3 or azure")
	ErrInvalidConfig     = errors.New("config is required")
	ErrBucketStoreNotFound = errors.New("bucket store not found")
	ErrBucketNameTaken   = errors.New("bucket name already registered")
)

// AzureConfig is the plaintext config shape for azure bucket stores.
type AzureConfig struct {
	ConnectionString string `json:"connection_string"`
}

// S3Config is the plaintext config shape for s3 bucket stores.
type S3Config struct {
	Server    string `json:"server"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

// BucketStoreDetails is the public API representation (config never included).
type BucketStoreDetails struct {
	ID         string    `json:"id"`
	BucketName string    `json:"bucket_name"`
	BucketType string    `json:"bucket_type"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateInput holds fields for creating a bucket store.
type CreateInput struct {
	BucketName string
	BucketType string
	Config     json.RawMessage
}

// UpdateInput holds fields for updating a bucket store.
type UpdateInput struct {
	ID         string
	BucketName string
	BucketType string
	Config     json.RawMessage // optional; empty means keep existing
}

// Service orchestrates bucket store business logic.
type Service struct {
	repo      *storage.BucketStoreRepository
	encryptor *crypto.SecretEncryptor
	logger    *slog.Logger
}

// ServiceConfig wires dependencies for Service.
type ServiceConfig struct {
	Repo      *storage.BucketStoreRepository
	Encryptor *crypto.SecretEncryptor
	Logger    *slog.Logger
}

// NewService constructs a bucket store Service.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Repo == nil {
		return nil, errors.New("bucket store repo is required")
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

// Create registers a new bucket store with encrypted config.
func (s *Service) Create(in CreateInput) (*BucketStoreDetails, error) {
	bucketType, err := normaliseBucketType(in.BucketType)
	if err != nil {
		return nil, err
	}
	if err := validateCreate(in, bucketType); err != nil {
		return nil, err
	}

	configJSON, err := validateAndMarshalConfig(bucketType, in.Config)
	if err != nil {
		return nil, err
	}

	encrypted, err := s.encryptor.Encrypt(string(configJSON))
	if err != nil {
		return nil, fmt.Errorf("encrypt config: %w", err)
	}

	now := time.Now().UTC()
	store := &storage.BucketStore{
		ID:              uuid.NewString(),
		BucketName:      in.BucketName,
		BucketType:      bucketType,
		ConfigEncrypted: encrypted,
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(store); err != nil {
		if errors.Is(err, storage.ErrBucketNameTaken) {
			return nil, ErrBucketNameTaken
		}
		return nil, fmt.Errorf("create bucket store: %w", err)
	}

	return toDetails(store), nil
}

// Get returns an active bucket store by ID.
func (s *Service) Get(id string) (*BucketStoreDetails, error) {
	store, err := s.repo.FindActiveByID(id)
	if err != nil {
		if errors.Is(err, storage.ErrBucketStoreNotFound) {
			return nil, ErrBucketStoreNotFound
		}
		return nil, fmt.Errorf("get bucket store: %w", err)
	}
	return toDetails(store), nil
}

// List returns all active bucket stores.
func (s *Service) List() ([]BucketStoreDetails, error) {
	stores, err := s.repo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("list bucket stores: %w", err)
	}

	out := make([]BucketStoreDetails, 0, len(stores))
	for i := range stores {
		out = append(out, *toDetails(&stores[i]))
	}
	return out, nil
}

// Update modifies an active bucket store. Config is re-encrypted when provided.
func (s *Service) Update(in UpdateInput) (*BucketStoreDetails, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, ErrBucketStoreNotFound
	}

	bucketType, err := normaliseBucketType(in.BucketType)
	if err != nil {
		return nil, err
	}
	if err := validateUpdate(in, bucketType); err != nil {
		return nil, err
	}

	store, err := s.repo.FindActiveByID(in.ID)
	if err != nil {
		if errors.Is(err, storage.ErrBucketStoreNotFound) {
			return nil, ErrBucketStoreNotFound
		}
		return nil, fmt.Errorf("find bucket store: %w", err)
	}

	if name := strings.TrimSpace(in.BucketName); name != "" && name != store.BucketName {
		existing, findErr := s.repo.FindByBucketName(name)
		if findErr == nil && existing.ID != store.ID {
			return nil, ErrBucketNameTaken
		}
		if findErr != nil && !errors.Is(findErr, storage.ErrBucketStoreNotFound) {
			return nil, fmt.Errorf("check bucket name: %w", findErr)
		}
	}

	store.BucketName = in.BucketName
	store.BucketType = bucketType
	store.UpdatedAt = time.Now().UTC()

	if len(in.Config) > 0 && string(in.Config) != "null" {
		configJSON, cfgErr := validateAndMarshalConfig(bucketType, in.Config)
		if cfgErr != nil {
			return nil, cfgErr
		}
		encrypted, encErr := s.encryptor.Encrypt(string(configJSON))
		if encErr != nil {
			return nil, fmt.Errorf("encrypt config: %w", encErr)
		}
		store.ConfigEncrypted = encrypted
	}

	if err := s.repo.Update(store); err != nil {
		if errors.Is(err, storage.ErrBucketNameTaken) {
			return nil, ErrBucketNameTaken
		}
		return nil, fmt.Errorf("update bucket store: %w", err)
	}

	return toDetails(store), nil
}

// Delete marks a bucket store inactive (soft delete).
func (s *Service) Delete(id string) error {
	if err := s.repo.MarkInactive(id); err != nil {
		if errors.Is(err, storage.ErrBucketStoreNotFound) {
			return ErrBucketStoreNotFound
		}
		return fmt.Errorf("delete bucket store: %w", err)
	}
	return nil
}

func normaliseBucketType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case storage.BucketTypeS3:
		return storage.BucketTypeS3, nil
	case storage.BucketTypeAzure, "azure_blob", "azure-blob":
		return storage.BucketTypeAzure, nil
	default:
		return "", ErrInvalidBucketType
	}
}

func validateCreate(in CreateInput, bucketType string) error {
	if strings.TrimSpace(in.BucketName) == "" {
		return ErrInvalidBucketName
	}
	if len(in.Config) == 0 || string(in.Config) == "null" {
		return ErrInvalidConfig
	}
	return validateConfigFields(bucketType, in.Config)
}

func validateUpdate(in UpdateInput, bucketType string) error {
	if strings.TrimSpace(in.BucketName) == "" {
		return ErrInvalidBucketName
	}
	if len(in.Config) == 0 || string(in.Config) == "null" {
		return nil
	}
	return validateConfigFields(bucketType, in.Config)
}

func validateConfigFields(bucketType string, raw json.RawMessage) error {
	switch bucketType {
	case storage.BucketTypeS3:
		var cfg S3Config
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("%w: invalid s3 config json", ErrInvalidConfig)
		}
		if strings.TrimSpace(cfg.Server) == "" ||
			strings.TrimSpace(cfg.AccessKey) == "" ||
			strings.TrimSpace(cfg.SecretKey) == "" {
			return fmt.Errorf("%w: s3 config requires server, access_key, and secret_key", ErrInvalidConfig)
		}
	case storage.BucketTypeAzure:
		var cfg AzureConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("%w: invalid azure config json", ErrInvalidConfig)
		}
		if strings.TrimSpace(cfg.ConnectionString) == "" {
			return fmt.Errorf("%w: azure config requires connection_string", ErrInvalidConfig)
		}
	}
	return nil
}

func validateAndMarshalConfig(bucketType string, raw json.RawMessage) ([]byte, error) {
	if err := validateConfigFields(bucketType, raw); err != nil {
		return nil, err
	}

	switch bucketType {
	case storage.BucketTypeS3:
		var cfg S3Config
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("%w: invalid s3 config json", ErrInvalidConfig)
		}
		return json.Marshal(cfg)
	case storage.BucketTypeAzure:
		var cfg AzureConfig
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("%w: invalid azure config json", ErrInvalidConfig)
		}
		return json.Marshal(cfg)
	default:
		return nil, ErrInvalidBucketType
	}
}

func toDetails(store *storage.BucketStore) *BucketStoreDetails {
	return &BucketStoreDetails{
		ID:         store.ID,
		BucketName: store.BucketName,
		BucketType: store.BucketType,
		CreatedAt:  store.CreatedAt,
		UpdatedAt:  store.UpdatedAt,
	}
}
