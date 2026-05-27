import { describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { makeCredentials } from "../helpers/factories";

describe("POST /forgot-password", () => {
  test("accepts valid email", async () => {
    const creds = makeCredentials();
    const registerResponse = await apiRequest("POST", "/register", { body: creds });
    expect([201, 409]).toContain(registerResponse.status);

    const response = await apiRequest("POST", "/forgot-password", {
      body: { email: creds.email },
    });
    expect(response.status).toBe(200);
    expect(response.body?.success).toBe(true);
  });

  const cases = [
    { name: "rejects empty payload", body: {}, expectedStatus: 400 },
    { name: "rejects invalid email", body: { email: "not-an-email" }, expectedStatus: 400 },
  ];

  for (const testCase of cases) {
    test(testCase.name, async () => {
      const response = await apiRequest("POST", "/forgot-password", {
        body: testCase.body,
      });
      expect(response.status).toBe(testCase.expectedStatus);
      expect(response.body?.error).toBeString();
    });
  }
});
