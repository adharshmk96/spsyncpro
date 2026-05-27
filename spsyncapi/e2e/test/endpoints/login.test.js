import { describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { makeCredentials } from "../helpers/factories";

describe("POST /login", () => {
  test("logs in registered user", async () => {
    const creds = makeCredentials();
    const registerResponse = await apiRequest("POST", "/register", { body: creds });
    expect([201, 409]).toContain(registerResponse.status);

    const response = await apiRequest("POST", "/login", { body: creds });
    expect(response.status).toBe(200);
    expect(response.body?.token).toBeString();
  });

  const failureCases = [
    {
      name: "rejects missing payload",
      body: {},
      expectedStatus: 400,
    },
    {
      name: "rejects invalid credentials",
      body: { email: "missing@example.com", password: "badpass123" },
      expectedStatus: 401,
    },
  ];

  for (const testCase of failureCases) {
    test(testCase.name, async () => {
      const response = await apiRequest("POST", "/login", { body: testCase.body });
      expect(response.status).toBe(testCase.expectedStatus);
      expect(response.body?.error).toBeString();
    });
  }
});
