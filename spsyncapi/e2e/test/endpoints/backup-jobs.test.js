import { beforeEach, describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { registerAndLogin } from "../helpers/auth";
import {
  makeBackupJobPayload,
  makeBucketStorePayload,
  makeOrganizationPayload,
  uniqueTag,
} from "../helpers/factories";

describe("backup-jobs endpoints", () => {
  let token = "";

  async function createOrgAndStore() {
    const orgResponse = await apiRequest("POST", "/organizations", {
      token,
      body: makeOrganizationPayload(),
    });
    expect(orgResponse.status).toBe(201);

    const storeResponse = await apiRequest("POST", "/bucket-stores", {
      token,
      body: makeBucketStorePayload(),
    });
    expect(storeResponse.status).toBe(201);

    return {
      organizationId: orgResponse.body.organization.id,
      bucketStoreId: storeResponse.body.bucket_store.id,
    };
  }

  beforeEach(async () => {
    const auth = await registerAndLogin();
    token = auth.token;
  });

  test("POST /backup-jobs creates job", async () => {
    const deps = await createOrgAndStore();
    const payload = makeBackupJobPayload(deps);
    const response = await apiRequest("POST", "/backup-jobs", { token, body: payload });
    expect(response.status).toBe(201);
    expect(response.body?.backup_job?.id).toBeString();
  });

  test("POST /backup-jobs rejects invalid payload", async () => {
    const response = await apiRequest("POST", "/backup-jobs", { token, body: {} });
    expect(response.status).toBe(400);
  });

  test("GET /backup-jobs lists jobs", async () => {
    const deps = await createOrgAndStore();
    await apiRequest("POST", "/backup-jobs", {
      token,
      body: makeBackupJobPayload(deps),
    });

    const response = await apiRequest("GET", "/backup-jobs", { token });
    expect(response.status).toBe(200);
    expect(Array.isArray(response.body?.backup_jobs)).toBe(true);
  });

  test("GET /backup-jobs/:id returns job", async () => {
    const deps = await createOrgAndStore();
    const create = await apiRequest("POST", "/backup-jobs", {
      token,
      body: makeBackupJobPayload(deps),
    });
    const id = create.body.backup_job.id;

    const response = await apiRequest("GET", `/backup-jobs/${id}`, { token });
    expect(response.status).toBe(200);
    expect(response.body?.backup_job?.id).toBe(id);
  });

  test("GET /backup-jobs/:id returns 404 for missing job", async () => {
    const response = await apiRequest("GET", "/backup-jobs/00000000-0000-0000-0000-000000000999", { token });
    expect(response.status).toBe(404);
  });

  test("PUT /backup-jobs/:id updates job", async () => {
    const deps = await createOrgAndStore();
    const create = await apiRequest("POST", "/backup-jobs", {
      token,
      body: makeBackupJobPayload(deps),
    });
    const id = create.body.backup_job.id;
    const payload = makeBackupJobPayload({ ...deps, tag: uniqueTag() });
    payload.active = false;

    const response = await apiRequest("PUT", `/backup-jobs/${id}`, {
      token,
      body: payload,
    });
    expect(response.status).toBe(200);
    expect(response.body?.backup_job?.id).toBe(id);
  });

  test("PUT /backup-jobs/:id returns 404 for missing job", async () => {
    const deps = await createOrgAndStore();
    const payload = makeBackupJobPayload(deps);
    const response = await apiRequest("PUT", "/backup-jobs/00000000-0000-0000-0000-000000000999", {
      token,
      body: payload,
    });
    expect(response.status).toBe(404);
  });

  test("DELETE /backup-jobs/:id soft deletes job", async () => {
    const deps = await createOrgAndStore();
    const create = await apiRequest("POST", "/backup-jobs", {
      token,
      body: makeBackupJobPayload(deps),
    });
    const id = create.body.backup_job.id;

    const del = await apiRequest("DELETE", `/backup-jobs/${id}`, { token });
    expect(del.status).toBe(200);
    expect(del.body?.success).toBe(true);

    const getAfterDelete = await apiRequest("GET", `/backup-jobs/${id}`, { token });
    expect(getAfterDelete.status).toBe(404);
  });

  test("GET /backup-jobs/:id returns 404 for another member's job", async () => {
    const userA = await registerAndLogin();
    const deps = await (async () => {
      const orgResponse = await apiRequest("POST", "/organizations", {
        token: userA.token,
        body: makeOrganizationPayload(),
      });
      const storeResponse = await apiRequest("POST", "/bucket-stores", {
        token: userA.token,
        body: makeBucketStorePayload(),
      });
      return {
        organizationId: orgResponse.body.organization.id,
        bucketStoreId: storeResponse.body.bucket_store.id,
      };
    })();
    const create = await apiRequest("POST", "/backup-jobs", {
      token: userA.token,
      body: makeBackupJobPayload(deps),
    });
    expect(create.status).toBe(201);

    const userB = await registerAndLogin();
    const response = await apiRequest("GET", `/backup-jobs/${create.body.backup_job.id}`, {
      token: userB.token,
    });
    expect(response.status).toBe(404);
  });

  test("POST /backup-jobs rejects another member's organization and bucket references", async () => {
    const userA = await registerAndLogin();
    const orgResponse = await apiRequest("POST", "/organizations", {
      token: userA.token,
      body: makeOrganizationPayload(),
    });
    const storeResponse = await apiRequest("POST", "/bucket-stores", {
      token: userA.token,
      body: makeBucketStorePayload(),
    });

    const userB = await registerAndLogin();
    const response = await apiRequest("POST", "/backup-jobs", {
      token: userB.token,
      body: makeBackupJobPayload({
        organizationId: orgResponse.body.organization.id,
        bucketStoreId: storeResponse.body.bucket_store.id,
      }),
    });
    expect(response.status).toBe(400);
  });
});
