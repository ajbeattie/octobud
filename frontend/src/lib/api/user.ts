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

import {
	fetchWithAuth,
	ApiUnreachableError,
	isNetworkError,
	isProxyConnectionError,
} from "./fetch";
import { getAuthToken, getCSRFToken } from "$lib/stores/authStore";

export interface LoginResponse {
	token: string;
	username: string;
	csrfToken: string;
}

export interface SyncSettings {
	initialSyncDays?: number | null;
	initialSyncMaxCount?: number | null;
	initialSyncUnreadOnly: boolean;
	setupCompleted: boolean;
}

export interface UserResponse {
	username: string;
	syncSettings?: SyncSettings | null;
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

	let response: Response;
	try {
		response = await fetchFn("/api/user/login", {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify({ username, password }),
			credentials: "include", // Include cookies for CSRF token
		});
	} catch (error) {
		// Network errors indicate the API is unreachable
		if (isNetworkError(error)) {
			throw new ApiUnreachableError();
		}
		throw error;
	}

	if (!response.ok) {
		// Check if this is a proxy connection error (backend unreachable)
		let responseText = "";
		let isValidJson = false;
		let apiErrorMessage = "";

		try {
			responseText = await response.text();
		} catch {
			// If we can't read the response, treat as proxy error
		}

		// Try to parse as JSON to get error message from backend
		if (responseText) {
			try {
				const errorData = JSON.parse(responseText);
				isValidJson = true;
				if (errorData.error) {
					apiErrorMessage = errorData.error;
				}
			} catch {
				// JSON parsing failed - might be proxy error HTML/text
			}
		}

		// Proxy errors: empty response, non-JSON response, or response with connection error keywords
		const isProxyError =
			(response.status === 500 || response.status === 502 || response.status === 503) &&
			(!responseText || !isValidJson || isProxyConnectionError(responseText));

		if (isProxyError) {
			throw new ApiUnreachableError();
		}

		// If we got a valid error message from the backend, use it
		if (apiErrorMessage) {
			throw new Error(apiErrorMessage);
		}

		throw new Error("Login failed");
	}

	return response.json();
}

export async function getCurrentUser(fetchImpl?: typeof fetch): Promise<UserResponse> {
	const response = await fetchWithAuth(
		"/api/user/me",
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

	const response = await fetchFn("/api/user/refresh", {
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

	const response = await fetchFn("/api/user/logout", {
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
		"/api/user/credentials",
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

export async function getSyncSettings(fetchImpl?: typeof fetch): Promise<SyncSettings> {
	const response = await fetchWithAuth(
		"/api/user/sync-settings",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get sync settings" }));
		throw new Error(error.error || "Failed to get sync settings");
	}

	return response.json();
}

export interface UpdateSyncSettingsRequest {
	initialSyncDays?: number | null;
	initialSyncMaxCount?: number | null;
	initialSyncUnreadOnly: boolean;
	setupCompleted: boolean;
}

export async function updateSyncSettings(
	settings: UpdateSyncSettingsRequest,
	fetchImpl?: typeof fetch
): Promise<SyncSettings> {
	const response = await fetchWithAuth(
		"/api/user/sync-settings",
		{
			method: "PUT",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(settings),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to update sync settings" }));
		throw new Error(error.error || "Failed to update sync settings");
	}

	return response.json();
}

export interface SyncState {
	oldestNotificationSyncedAt?: string | null;
	initialSyncCompletedAt?: string | null;
}

export async function getSyncState(fetchImpl?: typeof fetch): Promise<SyncState> {
	const response = await fetchWithAuth(
		"/api/user/sync-state",
		{
			method: "GET",
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Failed to get sync state" }));
		throw new Error(error.error || "Failed to get sync state");
	}

	return response.json();
}

export interface SyncOlderRequest {
	days: number;
	maxCount?: number | null;
	unreadOnly?: boolean;
	beforeDate?: string | null; // RFC3339 format date to override the "before" cutoff
}

export async function syncOlderNotifications(
	params: SyncOlderRequest,
	fetchImpl?: typeof fetch
): Promise<void> {
	const response = await fetchWithAuth(
		"/api/user/sync-older",
		{
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(params),
		},
		fetchImpl
	);

	if (!response.ok) {
		const error = await response
			.json()
			.catch(() => ({ error: "Failed to start sync of older notifications" }));
		throw new Error(error.error || "Failed to start sync of older notifications");
	}

	// Response is 202 Accepted with no body
}
