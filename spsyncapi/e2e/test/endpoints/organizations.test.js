import { beforeEach, describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { registerAndLogin } from "../helpers/auth";
import { createState } from "../helpers/state";
import { makeOrganizationPayload, uniqueTag } from "../helpers/factories";

describe("organizations endpoints", () => {
  let token = "";
  let state = createState();

  beforeEach(async () => {
    const auth = await registerAndLogin();
    token = auth.token;
    state = createState();
  });

  test("POST /organizations creates organization", async () => {
    const payload = makeOrganizationPayload();
    const response = await apiRequest("POST", "/organizations", { token, body: payload });
    expect(response.status).toBe(201);
    expect(response.body?.organization?.id).toBeString();
    state.organizationId = response.body.organization.id;
  });

  test("POST /organizations rejects invalid payload", async () => {
    const response = await apiRequest("POST", "/organizations", { token, body: {} });
    expect(response.status).toBe(400);
  });

  test("GET /organizations lists organizations", async () => {
    const payload = makeOrganizationPayload();
    await apiRequest("POST", "/organizations", { token, body: payload });
    const response = await apiRequest("GET", "/organizations", { token });
    expect(response.status).toBe(200);
    expect(Array.isArray(response.body?.organizations)).toBe(true);
  });

  test("GET /organizations/:id returns organization", async () => {
    const create = await apiRequest("POST", "/organizations", {
      token,
      body: makeOrganizationPayload(),
    });
    state.organizationId = create.body.organization.id;
    const response = await apiRequest("GET", `/organizations/${state.organizationId}`, { token });
    expect(response.status).toBe(200);
    expect(response.body?.organization?.id).toBe(state.organizationId);
  });

  const getCases = [
    { name: "returns 404 for missing organization", id: "00000000-0000-0000-0000-000000000999", expectedStatus: 404 },
  ];
  for (const testCase of getCases) {
    test(`GET /organizations/:id ${testCase.name}`, async () => {
      const response = await apiRequest("GET", `/organizations/${testCase.id}`, { token });
      expect(response.status).toBe(testCase.expectedStatus);
    });
  }

  test("PUT /organizations/:id updates organization", async () => {
    const create = await apiRequest("POST", "/organizations", {
      token,
      body: makeOrganizationPayload(),
    });
    state.organizationId = create.body.organization.id;
    const updatePayload = makeOrganizationPayload(uniqueTag());
    const response = await apiRequest("PUT", `/organizations/${state.organizationId}`, {
      token,
      body: updatePayload,
    });
    expect(response.status).toBe(200);
    expect(response.body?.organization?.name).toBe(updatePayload.name);
  });

  test("PUT /organizations/:id returns 404 for missing organization", async () => {
    const response = await apiRequest("PUT", "/organizations/00000000-0000-0000-0000-000000000999", {
      token,
      body: makeOrganizationPayload(),
    });
    expect(response.status).toBe(404);
  });

  test("DELETE /organizations/:id soft deletes organization", async () => {
    const create = await apiRequest("POST", "/organizations", {
      token,
      body: makeOrganizationPayload(),
    });
    const organizationId = create.body.organization.id;
    const del = await apiRequest("DELETE", `/organizations/${organizationId}`, { token });
    expect(del.status).toBe(200);
    expect(del.body?.success).toBe(true);

    const getAfterDelete = await apiRequest("GET", `/organizations/${organizationId}`, { token });
    expect(getAfterDelete.status).toBe(404);
  });

  test("GET /organizations/:id returns 404 for another member's organization", async () => {
    const userA = await registerAndLogin();
    const create = await apiRequest("POST", "/organizations", {
      token: userA.token,
      body: makeOrganizationPayload(),
    });
    expect(create.status).toBe(201);

    const userB = await registerAndLogin();
    const response = await apiRequest("GET", `/organizations/${create.body.organization.id}`, {
      token: userB.token,
    });
    expect(response.status).toBe(404);
  });

  test("GET /organizations excludes another member's organizations", async () => {
    const userA = await registerAndLogin();
    await apiRequest("POST", "/organizations", {
      token: userA.token,
      body: makeOrganizationPayload(),
    });

    const userB = await registerAndLogin();
    const response = await apiRequest("GET", "/organizations", { token: userB.token });
    expect(response.status).toBe(200);
    expect(response.body.organizations).toHaveLength(0);
  });
});
