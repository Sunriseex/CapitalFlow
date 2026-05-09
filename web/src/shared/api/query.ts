import type { QueryClient } from "@tanstack/react-query";
import { ApiClientError } from "../../api/client";

export function invalidateMoney(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: ["accounts"] });
  void queryClient.invalidateQueries({ queryKey: ["transactions"] });
  void queryClient.invalidateQueries({ queryKey: ["dashboard"] });
  void queryClient.invalidateQueries({ queryKey: ["balance"] });
  void queryClient.invalidateQueries({ queryKey: ["interest-rules"] });
}

export function errorMessage(err: unknown) {
  if (err instanceof ApiClientError) {
    return `${err.code ? `${err.code}: ` : ""}${err.message}`;
  }
  if (err instanceof Error) {
    return err.message;
  }
  return "Request failed";
}

