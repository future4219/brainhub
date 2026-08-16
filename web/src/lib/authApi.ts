import type { User } from "@/entities/user/entity";
import { APIError, postJSON, requestJSON } from "@/lib/api";

export async function getCurrentUser(): Promise<User | null> {
  try {
    return await requestJSON<User>("/me");
  } catch (error) {
    if (error instanceof APIError && error.status === 401) return null;
    throw error;
  }
}

export function register(input: { email: string; password: string; name: string }): Promise<User> {
  return postJSON("/auth/register", input);
}

export function login(input: { email: string; password: string }): Promise<User> {
  return postJSON("/auth/login", input);
}

export function logout(): Promise<void> {
  return postJSON("/auth/logout");
}
