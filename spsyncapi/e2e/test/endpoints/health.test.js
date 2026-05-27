import { describe, expect, test } from "bun:test";
import { apiRequest } from "../helpers/http";

describe("GET /health", () => {
  const cases = [{ name: "returns healthy status", expectedStatus: 200 }];

  for (const testCase of cases) {
    test(testCase.name, async () => {
      const response = await apiRequest("GET", "/health");
      expect(response.status).toBe(testCase.expectedStatus);
      expect(response.body?.status).toBe("ok");
    });
  }
});
