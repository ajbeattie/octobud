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

import { writable, derived, get } from "svelte/store";
import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import {
	refreshToken as refreshTokenAPI,
	updateCredentials as updateCredentialsAPI,
	logout as logoutAPI,
} from "$lib/api/auth";

const AUTH_TOKEN_KEY = "auth_token";
const AUTH_USERNAME_KEY = "auth_username";
const CSRF_TOKEN_KEY = "csrf_token";
const DB_NAME = "OctobudDB";
const DB_VERSION = 4; // Must match service worker DB_VERSION

// Internal stores
const tokenStore = writable<string | null>(null);
const usernameStore = writable<string | null>(null);
const csrfTokenStore = writable<string | null>(null);

// Initialize from localStorage if in browser
if (browser) {
	const storedToken = localStorage.getItem(AUTH_TOKEN_KEY);
	const storedUsername = localStorage.getItem(AUTH_USERNAME_KEY);
	const storedCSRFToken = localStorage.getItem(CSRF_TOKEN_KEY);
	if (storedToken) {
		tokenStore.set(storedToken);
	}
	if (storedUsername) {
		usernameStore.set(storedUsername);
	}
	if (storedCSRFToken) {
		csrfTokenStore.set(storedCSRFToken);
	}
}

// Derived stores
export const isAuthenticated = derived(tokenStore, ($token) => $token !== null);
export const currentUser = derived(usernameStore, ($username) => $username);

// Get auth token (for API calls)
export function getAuthToken(): string | null {
	return get(tokenStore);
}

// Get CSRF token (for API calls)
export function getCSRFToken(): string | null {
	return get(csrfTokenStore);
}

// Helper to save auth token to IndexedDB (for service worker access)
async function saveAuthToIndexedDB(token: string | null, username: string | null): Promise<void> {
	if (!browser) return;

	try {
		const db = await new Promise<IDBDatabase>((resolve, reject) => {
			const request = indexedDB.open(DB_NAME, DB_VERSION);

			request.onerror = () => reject(request.error);
			request.onsuccess = () => resolve(request.result);

			request.onupgradeneeded = (event) => {
				const db = (event.target as IDBOpenDBRequest).result;
				if (!db.objectStoreNames.contains("auth")) {
					db.createObjectStore("auth", { keyPath: "id" });
				}
			};
		});

		// Check if object store exists before trying to use it
		if (!db.objectStoreNames.contains("auth")) {
			console.warn("[Auth] Auth object store does not exist in IndexedDB");
			return;
		}

		return new Promise((resolve, reject) => {
			const tx = db.transaction("auth", "readwrite");
			const store = tx.objectStore("auth");
			const request = store.put({
				id: "token",
				token: token,
				username: username,
				timestamp: Date.now(),
			});

			request.onsuccess = () => resolve();
			request.onerror = () => reject(request.error);
		});
	} catch (error) {
		console.error("[Auth] Failed to save to IndexedDB:", error);
		// Don't throw - IndexedDB is optional, localStorage is primary
	}
}

// Helper to remove auth token from IndexedDB
async function removeAuthFromIndexedDB(): Promise<void> {
	if (!browser) return;

	try {
		const db = await new Promise<IDBDatabase>((resolve, reject) => {
			const request = indexedDB.open(DB_NAME, DB_VERSION);

			request.onerror = () => reject(request.error);
			request.onsuccess = () => resolve(request.result);

			request.onupgradeneeded = (event) => {
				const db = (event.target as IDBOpenDBRequest).result;
				if (!db.objectStoreNames.contains("auth")) {
					db.createObjectStore("auth", { keyPath: "id" });
				}
			};
		});

		// Check if object store exists before trying to use it
		if (!db.objectStoreNames.contains("auth")) {
			console.warn("[Auth] Auth object store does not exist in IndexedDB");
			return;
		}

		return new Promise((resolve, reject) => {
			const tx = db.transaction("auth", "readwrite");
			const store = tx.objectStore("auth");
			const request = store.delete("token");

			request.onsuccess = () => resolve();
			request.onerror = () => reject(request.error);
		});
	} catch (error) {
		console.error("[Auth] Failed to remove from IndexedDB:", error);
		// Don't throw - IndexedDB is optional
	}
}

// Login function
export async function login(username: string, password: string): Promise<void> {
	const response = await fetch("/api/auth/login", {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
		},
		body: JSON.stringify({ username, password }),
		credentials: "include", // Include cookies for CSRF token
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: "Login failed" }));
		throw new Error(error.error || "Login failed");
	}

	const data = await response.json();

	// Clear any existing auth state before setting new tokens
	if (browser) {
		localStorage.removeItem(AUTH_TOKEN_KEY);
		localStorage.removeItem(AUTH_USERNAME_KEY);
		localStorage.removeItem(CSRF_TOKEN_KEY);
		await removeAuthFromIndexedDB();
	}
	tokenStore.set(null);
	usernameStore.set(null);
	csrfTokenStore.set(null);

	// Set new auth state
	if (browser) {
		localStorage.setItem(AUTH_TOKEN_KEY, data.token);
		localStorage.setItem(AUTH_USERNAME_KEY, data.username);
		if (data.csrfToken) {
			localStorage.setItem(CSRF_TOKEN_KEY, data.csrfToken);
		}
		// Also save to IndexedDB for service worker access
		await saveAuthToIndexedDB(data.token, data.username);
	}

	tokenStore.set(data.token);
	usernameStore.set(data.username);
	if (data.csrfToken) {
		csrfTokenStore.set(data.csrfToken);
	}
}

// Logout function
let isLoggingOut = false; // Prevent infinite loops

export async function logout(): Promise<void> {
	// Prevent infinite loops - if already logging out, just clear local state
	if (isLoggingOut) {
		if (browser) {
			localStorage.removeItem(AUTH_TOKEN_KEY);
			localStorage.removeItem(AUTH_USERNAME_KEY);
			localStorage.removeItem(CSRF_TOKEN_KEY);
			await removeAuthFromIndexedDB();
		}
		tokenStore.set(null);
		usernameStore.set(null);
		csrfTokenStore.set(null);
		await goto(resolve("/login" as any));
		return;
	}

	isLoggingOut = true;
	try {
		// Call backend logout endpoint to revoke token
		await logoutAPI();
	} catch (err) {
		// Even if backend logout fails, clear local state
		console.error("Logout API call failed:", err);
	} finally {
		isLoggingOut = false;
	}

	if (browser) {
		localStorage.removeItem(AUTH_TOKEN_KEY);
		localStorage.removeItem(AUTH_USERNAME_KEY);
		localStorage.removeItem(CSRF_TOKEN_KEY);
		// Also remove from IndexedDB
		await removeAuthFromIndexedDB();
	}
	tokenStore.set(null);
	usernameStore.set(null);
	csrfTokenStore.set(null);
	await goto(resolve("/login" as any));
}

// Update credentials function
export async function updateCredentials(
	username: string | null,
	password: string | null,
	currentPassword: string
): Promise<void> {
	const data = await updateCredentialsAPI(username, password, currentPassword);

	if (browser) {
		localStorage.setItem(AUTH_USERNAME_KEY, data.username);
		// Update IndexedDB with new username (token stays the same)
		const currentToken = getAuthToken();
		await saveAuthToIndexedDB(currentToken, data.username);
	}

	usernameStore.set(data.username);
}

// Refresh token function - gets a new token with extended expiration
let isRefreshing = false;
let refreshPromise: Promise<void> | null = null;

export async function refreshToken(): Promise<void> {
	// Prevent concurrent refresh attempts
	if (isRefreshing && refreshPromise) {
		return refreshPromise;
	}

	const token = getAuthToken();
	if (!token) {
		throw new Error("Not authenticated");
	}

	isRefreshing = true;
	refreshPromise = (async () => {
		try {
			const data = await refreshTokenAPI();

			if (browser) {
				localStorage.setItem(AUTH_TOKEN_KEY, data.token);
				localStorage.setItem(AUTH_USERNAME_KEY, data.username);
				if (data.csrfToken) {
					localStorage.setItem(CSRF_TOKEN_KEY, data.csrfToken);
				}
				// Also update IndexedDB for service worker access
				await saveAuthToIndexedDB(data.token, data.username);
			}

			tokenStore.set(data.token);
			usernameStore.set(data.username);
			if (data.csrfToken) {
				csrfTokenStore.set(data.csrfToken);
			}
		} catch (err) {
			// If refresh fails, clear auth state
			if (browser) {
				localStorage.removeItem(AUTH_TOKEN_KEY);
				localStorage.removeItem(AUTH_USERNAME_KEY);
				localStorage.removeItem(CSRF_TOKEN_KEY); // Clear CSRF token too
				await removeAuthFromIndexedDB();
			}
			tokenStore.set(null);
			usernameStore.set(null);
			csrfTokenStore.set(null);
			throw err;
		} finally {
			isRefreshing = false;
			refreshPromise = null;
		}
	})();

	return refreshPromise;
}
