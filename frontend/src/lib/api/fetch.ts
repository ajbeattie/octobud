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

import { getAuthToken, getCSRFToken, logout, refreshToken } from "$lib/stores/authStore";
import { goto } from "$app/navigation";
import { browser } from "$app/environment";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";

/**
 * Custom error class for when the API is unreachable (network errors).
 * This is different from API errors (which have HTTP status codes).
 */
export class ApiUnreachableError extends Error {
	constructor(message: string = "Unable to reach the API server") {
		super(message);
		this.name = "ApiUnreachableError";
	}
}

/**
 * Check if an error is a network error (API unreachable).
 * This includes TypeError from fetch failures and other network-related errors.
 */
export function isNetworkError(error: unknown): boolean {
	if (error instanceof ApiUnreachableError) {
		return true;
	}
	if (error instanceof TypeError) {
		// TypeError with "Failed to fetch" indicates network failure
		return error.message.includes("Failed to fetch") || error.message.includes("NetworkError");
	}
	// Check for other network-related error messages
	if (error instanceof Error) {
		const msg = error.message.toLowerCase();
		return (
			msg.includes("network") ||
			msg.includes("connection") ||
			msg.includes("econnrefused") ||
			msg.includes("unreachable") ||
			msg.includes("proxy error") ||
			msg.includes("bad gateway") ||
			msg.includes("service unavailable")
		);
	}
	return false;
}

/**
 * Check if an error message indicates the API is unreachable.
 * This catches proxy errors that return 500/502/503 when the backend is down.
 */
export function isProxyConnectionError(errorMessage: string): boolean {
	const msg = errorMessage.toLowerCase();
	return (
		msg.includes("econnrefused") ||
		msg.includes("proxy error") ||
		msg.includes("bad gateway") ||
		msg.includes("service unavailable") ||
		msg.includes("connection refused") ||
		msg.includes("connect econnrefused") ||
		msg.includes("unable to connect") ||
		msg.includes("socket hang up") ||
		msg.includes("etimedout") ||
		msg.includes("ehostunreach")
	);
}

export function buildApiUrl(path: string): string {
	if (!API_BASE_URL) {
		return path;
	}
	const base = API_BASE_URL.replace(/\/+$/, "");
	return `${base}${path}`;
}

export async function fetchWithAuth(
	url: string,
	options: RequestInit = {},
	fetchImpl?: typeof fetch
): Promise<Response> {
	const fetchFn = fetchImpl || fetch;
	const token = getAuthToken();

	const headers = new Headers(options.headers);
	if (token) {
		headers.set("Authorization", `Bearer ${token}`);
	}

	// Add CSRF token for state-changing requests (POST, PUT, DELETE, PATCH)
	const method = options.method?.toUpperCase() || "GET";
	if (method !== "GET" && method !== "HEAD" && method !== "OPTIONS") {
		const csrfToken = getCSRFToken();
		if (csrfToken) {
			headers.set("X-CSRF-Token", csrfToken);
		}
	}

	let response: Response;
	try {
		response = await fetchFn(buildApiUrl(url), {
			...options,
			headers,
			credentials: "include", // Include cookies for CSRF token
		});
	} catch (error) {
		// Network errors (TypeError: Failed to fetch) indicate the API is unreachable
		if (isNetworkError(error)) {
			throw new ApiUnreachableError();
		}
		throw error;
	}

	// Handle 401 Unauthorized - try to refresh token first
	// Handle 403 Forbidden - CSRF token issue, try to refresh token
	if ((response.status === 401 || response.status === 403) && browser) {
		// Don't retry if this is already a logout or refresh request (avoid infinite loops)
		if (url.includes("/user/logout") || url.includes("/user/refresh")) {
			if (response.status === 401) {
				// For logout/refresh 401, just clear local state and redirect
				await logout();
				throw new Error("Unauthorized");
			}
			// For 403 on logout/refresh, return the response (don't retry)
			return response;
		}

		try {
			// Attempt to refresh the token (this will handle concurrent calls)
			await refreshToken();
			// Retry the original request with the new token
			const newToken = getAuthToken();
			const retryHeaders = new Headers(options.headers);
			if (newToken) {
				retryHeaders.set("Authorization", `Bearer ${newToken}`);
			}
			// Add CSRF token for retry if it's a state-changing request
			const retryMethod = options.method?.toUpperCase() || "GET";
			if (retryMethod !== "GET" && retryMethod !== "HEAD" && retryMethod !== "OPTIONS") {
				const csrfToken = getCSRFToken();
				if (csrfToken) {
					retryHeaders.set("X-CSRF-Token", csrfToken);
				}
			}
			let retryResponse: Response;
			try {
				retryResponse = await fetchFn(buildApiUrl(url), {
					...options,
					headers: retryHeaders,
					credentials: "include", // Include cookies for CSRF token
				});
			} catch (retryError) {
				// Network errors during retry also indicate API is unreachable
				if (isNetworkError(retryError)) {
					throw new ApiUnreachableError();
				}
				throw retryError;
			}
			// If retry still fails, logout
			if (retryResponse.status === 401 || retryResponse.status === 403) {
				await logout();
				throw new Error("Unauthorized");
			}
			return retryResponse;
		} catch (refreshError) {
			// Refresh failed, logout and redirect
			// Only logout if we're not already in a logout flow
			if (!url.includes("/user/logout")) {
				await logout();
			}
			throw new Error("Unauthorized");
		}
	}

	return response;
}
