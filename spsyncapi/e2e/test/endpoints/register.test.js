import { describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";
import { makeCredentials } from "../helpers/factories";

describe("POST /register", () => {
  const successCase = [{ name: "registers a new user", expectedStatus: 201 }];

  for (const testCase of successCase) {
    test(testCase.name, async () => {
      const payload = makeCredentials();
      const response = await apiRequest("POST", "/register", { body: payload });
      expect(response.status).toBe(testCase.expectedStatus);
      expect(response.body?.token).toBeString();
    });
  }

  const failureCases = [
    {
      name: "fails when payload is missing",
      body: {},
      expectedStatus: 400,
    },
    {
      name: "fails for duplicate email",
      expectedStatus: 409,
      buildBody: () => makeCredentials(),
    },
  ];

  for (const testCase of failureCases) {
    test(testCase.name, async () => {
      if (testCase.buildBody) {
        const creds = testCase.buildBody();
        const first = await apiRequest("POST", "/register", { body: creds });
        expect([201, 409]).toContain(first.status);
        const second = await apiRequest("POST", "/register", { body: creds });
        expect(second.status).toBe(testCase.expectedStatus);
        return;
      }

      const response = await apiRequest("POST", "/register", { body: testCase.body });
      expect(response.status).toBe(testCase.expectedStatus);
    });
  }
});
