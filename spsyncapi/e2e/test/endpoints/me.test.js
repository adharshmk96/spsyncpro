import { describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { registerAndLogin } from "../helpers/auth";

describe("GET /me", () => {
  const cases = [
    { name: "rejects missing token", token: null, expectedStatus: 401 },
    { name: "returns profile with valid token", token: "auth", expectedStatus: 200 },
  ];

  for (const testCase of cases) {
    test(testCase.name, async () => {
      const context = testCase.token === "auth" ? await registerAndLogin() : null;
      const response = await apiRequest("GET", "/me", { token: context?.token });
      expect(response.status).toBe(testCase.expectedStatus);
      if (testCase.expectedStatus === 200) {
        expect(response.body?.user?.email).toBe(context.email);
      } else {
        expect(response.body?.error).toBeString();
      }
    });
  }
});
