import { apiGet, apiPost } from "./client";
import type { User } from "./types";

/** The signed-in user, or null when nobody is signed in. */
export const getMe = () => apiGet<{ user: User | null }>("/auth/me").then((r) => r.user);

export const login = (email: string, password: string) => apiPost<User>("/auth/login", { email, password });

export const register = (email: string, password: string, display_name: string) =>
  apiPost<User>("/auth/register", { email, password, display_name });

export const logout = () => apiPost<null>("/auth/logout");
