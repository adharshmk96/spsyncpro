import { beforeEach, describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { registerAndLogin } from "../helpers/auth";
import { makeBucketStorePayload, uniqueTag } from "../helpers/factories";

describe("bucket-stores endpoints", () => {
  let token = "";

  beforeEach(async () => {
    const auth = await registerAndLogin();
    token = auth.token;
  });

  test("POST /bucket-stores creates store", async () => {
    const payload = makeBucketStorePayload();
    const response = await apiRequest("POST", "/bucket-stores", { token, body: payload });
    expect(response.status).toBe(201);
    expect(response.body?.bucket_store?.id).toBeString();
  });

  test("POST /bucket-stores rejects invalid payload", async () => {
    const response = await apiRequest("POST", "/bucket-stores", { token, body: {} });
    expect(response.status).toBe(400);
  });

  test("GET /bucket-stores lists stores", async () => {
    await apiRequest("POST", "/bucket-stores", { token, body: makeBucketStorePayload() });
    const response = await apiRequest("GET", "/bucket-stores", { token });
    expect(response.status).toBe(200);
    expect(Array.isArray(response.body?.bucket_stores)).toBe(true);
  });

  test("GET /bucket-stores/:id returns store", async () => {
    const create = await apiRequest("POST", "/bucket-stores", {
      token,
      body: makeBucketStorePayload(),
    });
    const id = create.body.bucket_store.id;
    const response = await apiRequest("GET", `/bucket-stores/${id}`, { token });
    expect(response.status).toBe(200);
    expect(response.body?.bucket_store?.id).toBe(id);
  });

  test("GET /bucket-stores/:id returns 404 for missing store", async () => {
    const response = await apiRequest("GET", "/bucket-stores/00000000-0000-0000-0000-000000000999", { token });
    expect(response.status).toBe(404);
  });

  test("PUT /bucket-stores/:id updates store", async () => {
    const create = await apiRequest("POST", "/bucket-stores", {
      token,
      body: makeBucketStorePayload(),
    });
    const id = create.body.bucket_store.id;
    const payload = makeBucketStorePayload(uniqueTag());
    const response = await apiRequest("PUT", `/bucket-stores/${id}`, { token, body: payload });
    expect(response.status).toBe(200);
    expect(response.body?.bucket_store?.bucket_name).toBe(payload.bucket_name);
  });

  test("PUT /bucket-stores/:id returns 404 for missing store", async () => {
    const response = await apiRequest("PUT", "/bucket-stores/00000000-0000-0000-0000-000000000999", {
      token,
      body: makeBucketStorePayload(),
    });
    expect(response.status).toBe(404);
  });

  test("DELETE /bucket-stores/:id soft deletes store", async () => {
    const create = await apiRequest("POST", "/bucket-stores", {
      token,
      body: makeBucketStorePayload(),
    });
    const id = create.body.bucket_store.id;
    const del = await apiRequest("DELETE", `/bucket-stores/${id}`, { token });
    expect(del.status).toBe(200);
    expect(del.body?.success).toBe(true);

    const getAfterDelete = await apiRequest("GET", `/bucket-stores/${id}`, { token });
    expect(getAfterDelete.status).toBe(404);
  });
});
