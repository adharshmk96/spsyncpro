import { describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { registerAndLogin } from "../helpers/auth";

describe("POST /change-password", () => {
  test("changes password with valid token and credentials", async () => {
    const context = await registerAndLogin();
    const response = await apiRequest("POST", "/change-password", {
      token: context.token,
      body: {
        current_password: context.password,
        new_password: "newpass123",
      },
    });
    expect(response.status).toBe(200);
    expect(response.body?.success).toBe(true);
  });

  const cases = [
    {
      name: "rejects missing token",
      token: null,
      body: { current_password: "password123", new_password: "newpass123" },
      expectedStatus: 401,
    },
    {
      name: "rejects invalid payload",
      token: "auth",
      body: {},
      expectedStatus: 400,
    },
  ];

  for (const testCase of cases) {
    test(testCase.name, async () => {
      const context = testCase.token === "auth" ? await registerAndLogin() : null;
      const response = await apiRequest("POST", "/change-password", {
        token: context?.token,
        body: testCase.body,
      });
      expect(response.status).toBe(testCase.expectedStatus);
      expect(response.body?.error).toBeString();
    });
  }
});
