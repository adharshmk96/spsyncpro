package organization_test

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"spsyncapi/internal/crypto"
	"spsyncapi/internal/organization"
	"spsyncapi/internal/storage"
)

const (
	testMemberA = "member-test-a"
	testMemberB = "member-test-b"
)

func newTestOrgService(t *testing.T) *organization.Service {
	t.Helper()

	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	enc, err := crypto.NewSecretEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	svc, err := organization.NewService(organization.ServiceConfig{
		Repo:      storage.NewOrganizationRepository(db),
		Encryptor: enc,
		Logger:    logger,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

func TestCreate_Get_List_Delete(t *testing.T) {
	svc := newTestOrgService(t)

	created, err := svc.Create(organization.CreateInput{
		MemberID:     testMemberA,
		Name:         "Acme Corp",
		TenantID:     "tenant-abc",
		ClientID:     "client-xyz",
		TenantSecret: "secret-value",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := svc.Get(testMemberA, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Fatalf("name: got %q", got.Name)
	}

	list, err := svc.List(testMemberA)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len: got %d", len(list))
	}

	if err := svc.Delete(testMemberA, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := svc.Get(testMemberA, created.ID); err == nil {
		t.Fatal("expected not found after delete")
	}

	list, err = svc.List(testMemberA)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

func TestCreate_DuplicateTenantID(t *testing.T) {
	svc := newTestOrgService(t)

	in := organization.CreateInput{
		MemberID:     testMemberA,
		Name:         "First",
		TenantID:     "dup-tenant",
		ClientID:     "client-1",
		TenantSecret: "secret-1",
	}
	if _, err := svc.Create(in); err != nil {
		t.Fatalf("first create: %v", err)
	}

	in.Name = "Second"
	in.ClientID = "client-2"
	in.TenantSecret = "secret-2"
	if _, err := svc.Create(in); err == nil {
		t.Fatal("expected duplicate tenant id error")
	}
}

func TestDuplicateTenantID_DifferentMembers(t *testing.T) {
	svc := newTestOrgService(t)

	base := organization.CreateInput{
		Name:         "Org",
		TenantID:     "shared-tenant",
		ClientID:     "client-1",
		TenantSecret: "secret-1",
	}

	base.MemberID = testMemberA
	if _, err := svc.Create(base); err != nil {
		t.Fatalf("member A create: %v", err)
	}

	base.MemberID = testMemberB
	base.ClientID = "client-2"
	base.TenantSecret = "secret-2"
	if _, err := svc.Create(base); err != nil {
		t.Fatalf("member B create with same tenant_id: %v", err)
	}
}

func TestMemberIsolation(t *testing.T) {
	svc := newTestOrgService(t)

	created, err := svc.Create(organization.CreateInput{
		MemberID:     testMemberA,
		Name:         "Private Org",
		TenantID:     "tenant-private",
		ClientID:     "client-1",
		TenantSecret: "secret-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := svc.Get(testMemberB, created.ID); !errors.Is(err, organization.ErrOrganizationNotFound) {
		t.Fatalf("expected not found for other member, got: %v", err)
	}

	list, err := svc.List(testMemberB)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("other member list should be empty, got %d", len(list))
	}

	if err := svc.Delete(testMemberB, created.ID); !errors.Is(err, organization.ErrOrganizationNotFound) {
		t.Fatalf("expected not found on delete by other member, got: %v", err)
	}
}

func TestUpdate_TenantSecretOptional(t *testing.T) {
	svc := newTestOrgService(t)

	created, err := svc.Create(organization.CreateInput{
		MemberID:     testMemberA,
		Name:         "Org",
		TenantID:     "tenant-1",
		ClientID:     "client-1",
		TenantSecret: "original-secret",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.Update(testMemberA, organization.UpdateInput{
		ID:       created.ID,
		Name:     "Org Renamed",
		TenantID: "tenant-1",
		ClientID: "client-1",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Org Renamed" {
		t.Fatalf("name: got %q", updated.Name)
	}
}
