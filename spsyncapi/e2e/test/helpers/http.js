export const API_BASE_URL =
  process.env.E2E_BASE_URL?.trim() || "http://localhost:8080/api/v1";

export async function apiRequest(method, path, options = {}) {
  const { token, body, headers = {} } = options;
  const requestHeaders = {
    Accept: "application/json",
    ...headers,
  };

  if (token) {
    requestHeaders.Authorization = `Bearer ${token}`;
  }
  if (body !== undefined) {
    requestHeaders["Content-Type"] = "application/json";
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers: requestHeaders,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const rawText = await response.text();
  let json = null;
  if (rawText.length > 0) {
    try {
      json = JSON.parse(rawText);
    } catch {
      json = { raw: rawText };
    }
  }

  return {
    status: response.status,
    body: json,
    rawText,
  };
}
