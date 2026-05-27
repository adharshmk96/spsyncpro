import { describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { registerAndLogin } from "../helpers/auth";

describe("POST /logout", () => {
  const cases = [
    { name: "rejects missing token", withAuth: false, expectedStatus: 401 },
    { name: "logs out authenticated user", withAuth: true, expectedStatus: 200 },
  ];

  for (const testCase of cases) {
    test(testCase.name, async () => {
      const context = testCase.withAuth ? await registerAndLogin() : null;
      const response = await apiRequest("POST", "/logout", { token: context?.token });
      expect(response.status).toBe(testCase.expectedStatus);
      if (testCase.expectedStatus === 200) {
        expect(response.body?.success).toBe(true);
      } else {
        expect(response.body?.error).toBeString();
      }
    });
  }
});
