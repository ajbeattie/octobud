// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { fetchWithAuth } from "./fetch";
import { getAuthToken, getCSRFToken } from "$lib/stores/authStore";

export interface LoginResponse {
	token: string;
	username: string;
	csrfToken: string;
}

export interface UserResponse {
	username: string;
}

export interface UpdateCredentialsRequest {
	currentPassword: string;
	newUsername?: string | null;
	newPassword?: string | null;
}

export async function login(
	username: string,
	password: string,
	fetchImpl?: typeof fetch
): Promise<LoginResponse> {
	const fetchFn = fetchImpl || fetch;
	const response = await fetchFn("/api/auth/login", {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
		},
		body: JSON.stringify({ username, password }),
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Login failed" }));
		throw new Error(error.error || "Login failed");
	}

	return response.json();
}

export async function getCurrentUser(fetchImpl?: typeof fetch): Promise<UserResponse> {
	const response = await fetchWithAuth(
		"/api/auth/me",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get user" }));
		throw new Error(error.error || "Failed to get user");
	}

	return response.json();
}

export async function refreshToken(fetchImpl?: typeof fetch): Promise<LoginResponse> {
	// Use direct fetch to avoid infinite loops in fetchWithAuth
	const fetchFn = fetchImpl || fetch;
	const token = getAuthToken();
	const csrfToken = getCSRFToken();

	const headers: HeadersInit = {
		"Content-Type": "application/json",
	};
	if (token) {
		headers["Authorization"] = `Bearer ${token}`;
	}
	if (csrfToken) {
		headers["X-CSRF-Token"] = csrfToken;
	}

	const response = await fetchFn("/api/auth/refresh", {
		method: "POST",
		headers,
		credentials: "include", // Include cookies
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to refresh token" }));
		throw new Error(error.error || "Failed to refresh token");
	}

	return response.json();
}

export async function logout(fetchImpl?: typeof fetch): Promise<void> {
	// Use direct fetch to avoid infinite loops in fetchWithAuth
	const fetchFn = fetchImpl || fetch;
	const token = getAuthToken();
	const csrfToken = getCSRFToken();

	const headers: HeadersInit = {
		"Content-Type": "application/json",
	};
	if (token) {
		headers["Authorization"] = `Bearer ${token}`;
	}
	if (csrfToken) {
		headers["X-CSRF-Token"] = csrfToken;
	}

	const response = await fetchFn("/api/auth/logout", {
		method: "POST",
		headers,
		credentials: "include", // Include cookies
	});

	if (!response.ok) {
		// Even if logout fails, we should still clear local state
		const error = await response.json().catch(() => ({ error: "Logout failed" }));
		throw new Error(error.error || "Logout failed");
	}
}

export async function updateCredentials(
	username: string | null,
	password: string | null,
	currentPassword: string,
	fetchImpl?: typeof fetch
): Promise<UserResponse> {
	const response = await fetchWithAuth(
		"/api/auth/credentials",
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({
				currentPassword,
				newUsername: username,
				newPassword: password,
			}),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Update failed" }));
		throw new Error(error.error || "Update failed");
	}

	return response.json();
}
