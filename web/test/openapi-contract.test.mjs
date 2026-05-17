import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import YAML from "yaml";

const openapi = YAML.parse(readFileSync(resolve("../docs/openapi.yaml"), "utf8"));

describe("OpenAPI contract", () => {
  it("covers auth and profile routes used by the frontend", () => {
    expect(Object.keys(openapi.paths)).toEqual(
      expect.arrayContaining([
        "/auth/status",
        "/auth/setup",
        "/auth/login",
        "/auth/refresh",
        "/auth/logout",
        "/settings/profile",
        "/auth/password",
        "/auth/sessions",
        "/auth/sessions/{id}",
      ]),
    );
  });

  it("generates auth and profile schemas used by the API client", () => {
    expect(openapi.components.schemas).toEqual(
      expect.objectContaining({
        AuthResponse: expect.any(Object),
        AuthStatusResponse: expect.any(Object),
        AuthSessionsResponse: expect.any(Object),
        ChangePasswordRequest: expect.any(Object),
        Profile: expect.any(Object),
        UpdateProfileRequest: expect.any(Object),
      }),
    );
  });
});
