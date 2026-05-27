import { expect } from "bun:test";
import { apiRequest } from "./http";
import { makeCredentials } from "./factories";

export async function registerUser(credentials) {
  return apiRequest("POST", "/register", { body: credentials });
}

export async function loginUser(credentials) {
  return apiRequest("POST", "/login", { body: credentials });
}

export async function registerAndLogin() {
  const credentials = makeCredentials();
  const registerResponse = await registerUser(credentials);
  expect([201, 409]).toContain(registerResponse.status);

  const loginResponse = await loginUser(credentials);
  expect(loginResponse.status).toBe(200);
  expect(loginResponse.body?.token).toBeString();

  return {
    ...credentials,
    token: loginResponse.body.token,
  };
}
