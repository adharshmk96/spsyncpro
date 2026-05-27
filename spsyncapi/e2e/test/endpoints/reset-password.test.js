import { describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { makeCredentials } from "../helpers/factories";

describe("POST /reset-password", () => {
  test("rejects invalid token for existing account", async () => {
    const creds = makeCredentials();
    const registerResponse = await apiRequest("POST", "/register", { body: creds });
    expect([201, 409]).toContain(registerResponse.status);

    const response = await apiRequest("POST", "/reset-password", {
      body: {
        email: creds.email,
        token: "invalid-token",
        password: "newpass123",
      },
    });

    expect(response.status).toBe(400);
    expect(response.body?.error).toBeString();
  });

  const cases = [{ name: "rejects empty payload", body: {}, expectedStatus: 400 }];

  for (const testCase of cases) {
    test(testCase.name, async () => {
      const response = await apiRequest("POST", "/reset-password", { body: testCase.body });
      expect(response.status).toBe(testCase.expectedStatus);
      expect(response.body?.error).toBeString();
    });
  }
});
