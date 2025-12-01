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

import { browser } from "$app/environment";
import { debugLog } from "$lib/utils/debug";

const SW_URL = "/sw.js";
const SW_UPDATE_INTERVAL = 60 * 60 * 1000; // Check for updates every hour

export interface ServiceWorkerRegistrationState {
	registration: ServiceWorkerRegistration | null;
	isSupported: boolean;
	isActive: boolean;
	error: string | null;
}

let registrationState: ServiceWorkerRegistrationState = {
	registration: null,
	isSupported: false,
	isActive: false,
	error: null,
};

const stateListeners = new Set<(state: ServiceWorkerRegistrationState) => void>();

export function subscribeToSWState(listener: (state: ServiceWorkerRegistrationState) => void) {
	stateListeners.add(listener);
	// Immediately call with current state
	listener(registrationState);

	return () => {
		stateListeners.delete(listener);
	};
}

function updateState(updates: Partial<ServiceWorkerRegistrationState>) {
	registrationState = { ...registrationState, ...updates };
	stateListeners.forEach((listener) => listener(registrationState));
}

export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
	if (!browser) {
		return null;
	}

	if (!("serviceWorker" in navigator)) {
		debugLog("[SW Registration] Service workers not supported");
		updateState({ isSupported: false, error: "Service workers not supported" });
		return null;
	}

	updateState({ isSupported: true });

	try {
		const registration = await navigator.serviceWorker.register(SW_URL, {
			scope: "/",
		});

		debugLog("[SW Registration] Service Worker registered:", registration.scope);
		updateState({ registration, isActive: !!navigator.serviceWorker.controller });

		// Sync debug flag with service worker
		syncDebugFlagWithSW();

		// Handle updates
		registration.addEventListener("updatefound", () => {
			const newWorker = registration.installing;
			if (!newWorker) return;

			newWorker.addEventListener("statechange", () => {
				if (newWorker.state === "installed" && navigator.serviceWorker.controller) {
					// New service worker available
					debugLog("[SW Registration] New service worker available");
					// Could show a toast or prompt to reload
					updateState({ isActive: false });
				} else if (newWorker.state === "activated") {
					debugLog("[SW Registration] New service worker activated");
					updateState({ isActive: true });
					// Sync debug flag after activation
					syncDebugFlagWithSW();
				}
			});
		});

		// Listen for controller change (when SW takes control)
		navigator.serviceWorker.addEventListener("controllerchange", () => {
			debugLog("[SW Registration] Service worker controller changed");
			updateState({ isActive: !!navigator.serviceWorker.controller });
			// Sync debug flag when controller changes
			syncDebugFlagWithSW();
		});

		// Check for updates periodically
		// In development, check more frequently (every 30 seconds)
		const updateIntervalMs = import.meta.env.DEV ? 30000 : SW_UPDATE_INTERVAL;
		const updateInterval = setInterval(() => {
			registration.update().catch((err) => {
				console.error("[SW Registration] Failed to check for updates:", err);
			});
		}, updateIntervalMs);

		// In development, also check for updates on page load
		if (import.meta.env.DEV) {
			registration.update().catch((err) => {
				console.error("[SW Registration] Failed to check for updates on load:", err);
			});
		}

		// Clean up interval on page unload
		if (browser) {
			window.addEventListener("beforeunload", () => {
				clearInterval(updateInterval);
			});
		}

		return registration;
	} catch (error) {
		console.error("[SW Registration] Service Worker registration failed:", error);
		updateState({ error: error instanceof Error ? error.message : "Registration failed" });
		return null;
	}
}

export async function unregisterServiceWorker(): Promise<boolean> {
	if (!browser || !("serviceWorker" in navigator)) {
		return false;
	}

	try {
		const registration = await navigator.serviceWorker.getRegistration();
		if (registration) {
			const success = await registration.unregister();
			if (success) {
				debugLog("[SW Registration] Service Worker unregistered");
				updateState({ registration: null, isActive: false });
			}
			return success;
		}
		return false;
	} catch (error) {
		console.error("[SW Registration] Failed to unregister:", error);
		return false;
	}
}

// Request notification permission
export async function requestNotificationPermission(): Promise<NotificationPermission> {
	if (!browser || !("Notification" in window)) {
		debugLog("[SW Registration] Notifications not supported in this environment");
		return "denied";
	}

	const currentPermission = Notification.permission;
	debugLog("[SW Registration] Current notification permission:", currentPermission);

	if (currentPermission === "granted") {
		// Also notify service worker of permission status
		await sendMessageToSW({ type: "NOTIFICATION_PERMISSION", permission: "granted" });
		return "granted";
	}

	if (currentPermission === "denied") {
		debugLog("[SW Registration] Notification permission is denied");
		await sendMessageToSW({ type: "NOTIFICATION_PERMISSION", permission: "denied" });
		return "denied";
	}

	try {
		debugLog("[SW Registration] Requesting notification permission...");
		const permission = await Notification.requestPermission();
		debugLog("[SW Registration] Notification permission result:", permission);

		// Notify service worker of permission status
		await sendMessageToSW({ type: "NOTIFICATION_PERMISSION", permission });

		return permission;
	} catch (error) {
		console.error("[SW Registration] Failed to request notification permission:", error);
		await sendMessageToSW({ type: "NOTIFICATION_PERMISSION", permission: "denied" });
		return "denied";
	}
}

// Get current notification permission
export function getNotificationPermission(): NotificationPermission {
	if (!browser || !("Notification" in window)) {
		return "denied";
	}
	return Notification.permission;
}

// Sync debug flag with service worker
function syncDebugFlagWithSW() {
	if (!browser || !("serviceWorker" in navigator) || !navigator.serviceWorker.controller) {
		return;
	}

	// Check if debug is enabled (check localStorage or dev mode)
	const DEBUG_KEY = "octobud:debug";
	let debugEnabled = false;

	if (import.meta.env.DEV) {
		debugEnabled = true;
	} else {
		try {
			debugEnabled = localStorage.getItem(DEBUG_KEY) === "true";
		} catch {
			// Ignore localStorage errors
		}
	}

	// Send debug flag to service worker
	sendMessageToSW({
		type: "DEBUG_ENABLED",
		enabled: debugEnabled,
	});
}

// Send message to service worker
export async function sendMessageToSW(message: any, transfer?: Transferable[]): Promise<void> {
	if (!browser || !("serviceWorker" in navigator) || !navigator.serviceWorker.controller) {
		return;
	}

	try {
		if (transfer) {
			navigator.serviceWorker.controller.postMessage(message, transfer);
		} else {
			navigator.serviceWorker.controller.postMessage(message);
		}
	} catch (error) {
		console.error("[SW Registration] Failed to send message to SW:", error);
	}
}

// Send notification setting to service worker
export async function sendNotificationSettingToSW(enabled: boolean): Promise<void> {
	await sendMessageToSW({
		type: "NOTIFICATION_SETTING_CHANGED",
		enabled,
	});
}

// Mark notifications as seen in service worker
// Only marks as seen when the page is actually visible to the user
// This prevents marking notifications as seen when the user is on a different workspace
// or the tab is hidden, allowing the service worker to show desktop notifications
export async function markNotificationsSeenInSW(notificationIds: string[]): Promise<void> {
	// Only mark as seen if the page is visible
	// This ensures we don't mark notifications as seen when the user isn't actively viewing
	// The service worker will handle showing desktop notifications for unseen items
	if (typeof document !== "undefined" && document.hidden) {
		debugLog(
			"[SW Registration] Skipping mark as seen - page is hidden (document.hidden:",
			document.hidden,
			")"
		);
		// Still send a message to wake up the SW and trigger its polling
		// This ensures the SW shows desktop notifications even if it was terminated
		await sendMessageToSW({ type: "FORCE_POLL" });
		return;
	}

	await sendMessageToSW({
		type: "MARK_SEEN",
		notificationIds,
	});
}

// Force service worker to poll immediately
export async function forceSWPoll(): Promise<void> {
	await sendMessageToSW({ type: "FORCE_POLL" });
}

// Send test notification request to service worker
export async function sendTestNotificationToSW(): Promise<void> {
	await sendMessageToSW({ type: "TEST_NOTIFICATION" });
}

// Get current registration state
export function getSWState(): ServiceWorkerRegistrationState {
	return registrationState;
}

// Force service worker to update immediately
export async function forceServiceWorkerUpdate(): Promise<void> {
	if (!browser || !("serviceWorker" in navigator)) {
		return;
	}

	try {
		const registration = await navigator.serviceWorker.getRegistration();
		if (registration) {
			console.log("[SW Registration] Forcing service worker update...");
			await registration.update();
			console.log("[SW Registration] Service worker update check completed");

			// If there's a waiting service worker, skip waiting and reload
			if (registration.waiting) {
				console.log("[SW Registration] New service worker waiting, activating...");
				registration.waiting.postMessage({ type: "SKIP_WAITING" });
				// The service worker will activate and we'll get a controllerchange event
			}
		}
	} catch (error) {
		console.error("[SW Registration] Failed to force update:", error);
	}
}

// Expose force update function to window for easy console access in dev
if (browser && import.meta.env.DEV) {
	(window as any).forceSWUpdate = forceServiceWorkerUpdate;
	console.log(
		"[SW Registration] Dev mode: Call forceSWUpdate() in console to update service worker"
	);
}
