import type { QueryClient } from "@tanstack/react-query";
import { ApiClientError } from "../../api/client";
import type { TranslationDictionary } from "../i18n/dictionaries/ru";

export function invalidateMoney(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: ["accounts"] });
  void queryClient.invalidateQueries({ queryKey: ["transactions"] });
  void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
  void queryClient.invalidateQueries({ queryKey: ["interest-rules"] });
}

type ErrorMessages = {
  requestFailed: string;
  apiRequestFailed: string;
  invalidApiResponse: string;
  loginRequired: string;
};

const defaultErrorMessages: ErrorMessages = {
  requestFailed: "Request failed",
  apiRequestFailed: "API request failed",
  invalidApiResponse: "Invalid API response",
  loginRequired: "Login required",
};

export function apiErrorMessages(t: TranslationDictionary): ErrorMessages {
  return {
    requestFailed: t.common.requestFailed,
    apiRequestFailed: t.common.apiRequestFailed,
    invalidApiResponse: t.common.invalidApiResponse,
    loginRequired: t.common.loginRequired,
  };
}

export function errorMessage(
  err: unknown,
  messages: ErrorMessages = defaultErrorMessages,
) {
  if (err instanceof ApiClientError) {
    if (err.code === "network_error") {
      return `${err.code}: ${messages.apiRequestFailed}`;
    }

    if (err.code === "invalid_response") {
      return `${err.code}: ${messages.invalidApiResponse}`;
    }

    if (err.code === "unauthorized") {
      return `${err.code}: ${messages.loginRequired}`;
    }

    return `${err.code ? `${err.code}: ` : ""}${err.message}`;
  }

  if (err instanceof Error) {
    return err.message;
  }

  return messages.requestFailed;
}
