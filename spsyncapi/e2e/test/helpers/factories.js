const nowStamp = Date.now();

export function uniqueTag() {
  return `${nowStamp}-${Math.random().toString(36).slice(2, 8)}`;
}

export function makeCredentials(tag = uniqueTag()) {
  return {
    email: `e2e-${tag}@example.com`,
    password: "password123",
  };
}

export function makeOrganizationPayload(tag = uniqueTag()) {
  return {
    name: `Acme ${tag}`,
    tenant_id: `tenant-${tag}`,
    client_id: `client-${tag}`,
    tenant_secret: `secret-${tag}`,
  };
}

export function makeBucketStorePayload(tag = uniqueTag()) {
  return {
    bucket_name: `bucket-${tag}`,
    bucket_type: "s3",
    config: {
      server: "https://s3.example.com",
      access_key: `access-${tag}`,
      secret_key: `secret-${tag}`,
    },
  };
}

export function makeBackupJobPayload({ organizationId, bucketStoreId, tag = uniqueTag() }) {
  const now = Date.now();
  return {
    active: true,
    start_at: new Date(now + 60_000).toISOString(),
    end_at: new Date(now + 3600_000).toISOString(),
    schedule: {
      interval: 300,
    },
    job_config: {
      organization: organizationId,
      bucket_store: bucketStoreId,
      share_point_site: `https://tenant.sharepoint.com/sites/${tag}`,
      filters: {
        document_libraries: ["Shared Documents"],
      },
    },
  };
}
