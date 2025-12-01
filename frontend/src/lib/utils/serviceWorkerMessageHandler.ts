// Copyright (C) Austin Beattie
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

import { get } from "svelte/store";
import { browser } from "$app/environment";
import { goto } from "$app/navigation";
import { resolve } from "$app/paths";
import { refreshToken } from "$lib/stores/authStore";
import { debugLog } from "$lib/utils/debug";
import { sendMessageToSW } from "$lib/utils/serviceWorkerRegistration";
import type { NotificationPageController } from "$lib/state/types";
import type { Readable } from "svelte/store";
import type { Notification } from "$lib/api/types";

interface ServiceWorkerHandlerParams {
	getPageController: () => NotificationPageController | null;
	getPageData: () => Readable<{ items: Notification[] }> | null;
}

/**
 * Set up all service worker event listeners.
 * This should be called early in the app lifecycle (e.g., in +layout.svelte)
 * to ensure messages aren't missed.
 *
 * Handles:
 * - Service worker messages (NEW_NOTIFICATIONS, OPEN_NOTIFICATION, TOKEN_EXPIRED)
 * - Page visibility changes (sync notifications and update SW sort date)
 *
 * @param params - Functions to get pageController and pageData when available
 * @returns Cleanup function to remove event listeners
 */
export function setupServiceWorkerHandlers(params: ServiceWorkerHandlerParams): () => void {
	if (!browser) {
		return () => {}; // No-op cleanup
	}

	const { getPageController, getPageData } = params;

	// Handle service worker messages
	const handleSWMessage = async (event: MessageEvent) => {
		debugLog("[App] Service worker message received:", event.data?.type, event.data);

		if (event.data?.type === "NEW_NOTIFICATIONS") {
			// Service worker detected new notifications
			// Trigger the existing sync handler to refresh the UI
			const pageController = getPageController();
			if (pageController) {
				void pageController.actions.handleSyncNewNotifications();
			}
		} else if (event.data?.type === "OPEN_NOTIFICATION") {
			// Service worker wants to open a specific notification
			// Navigate to the URL - the page's reactive statement will handle opening the detail
			const notificationId = event.data.notificationId;
			debugLog("[App] Received OPEN_NOTIFICATION message:", { notificationId });

			if (notificationId) {
				const notificationIdStr = String(notificationId);
				const targetUrl = `/views/inbox?id=${encodeURIComponent(notificationIdStr)}`;

				// Navigate to the URL - the reactive statement will handle opening the detail
				const urlObj = new URL(targetUrl, window.location.origin);
				const resolvedPath = resolve(urlObj.pathname as any) + urlObj.search;
				debugLog("[App] Navigating to:", resolvedPath);

				// Use goto() to navigate - this should update the URL and trigger reactive statements
				// If we're already on the same route, goto() should still update the query params
				// Use replaceState: true to avoid creating a new history entry
				void goto(resolvedPath, {
					noScroll: true,
					keepFocus: true,
					replaceState: true,
					invalidateAll: false,
				}).catch((error) => {
					console.error("[App] Failed to navigate to notification:", error);
				});
			}
		} else if (event.data?.type === "TOKEN_EXPIRED") {
			// Service worker detected expired token - try to refresh
			try {
				await refreshToken();
				debugLog("[App] Token refreshed successfully after SW notification");
			} catch (error) {
				console.error("[App] Failed to refresh token after SW notification:", error);
			}
		}
	};

	// Handle page visibility changes to refresh when page wakes up
	// This ensures we load latest notifications when returning to a suspended tab
	const handleVisibilityChange = async () => {
		if (!document.hidden) {
			const pageController = getPageController();
			const pageDataStore = getPageData();

			if (pageController && pageDataStore) {
				// First, sync notifications to get the latest state
				await pageController.actions.handleSyncNewNotifications();

				// After syncing, wait 200ms to allow the page to render with the updated state
				// Then update the service worker's sort date to prevent showing desktop notifications
				// for items the user has already seen.
				setTimeout(() => {
					const currentItems = get(pageDataStore).items;
					if (currentItems.length > 0) {
						// Find the maximum effectiveSortDate from the synced page data
						let maxEffectiveSortDate: string | null = null;
						for (const notification of currentItems) {
							const sortDate = notification.effectiveSortDate;
							if (sortDate) {
								if (!maxEffectiveSortDate || sortDate > maxEffectiveSortDate) {
									maxEffectiveSortDate = sortDate;
								}
							}
						}

						// Update service worker's sort date if we found one
						if (maxEffectiveSortDate) {
							debugLog(
								"[App] Window visible - updating SW sort date after sync to:",
								maxEffectiveSortDate
							);
							void sendMessageToSW({
								type: "UPDATE_SORT_DATE",
								effectiveSortDate: maxEffectiveSortDate,
							});
						}
					}
				}, 200);
			}
		}
	};

	// Set up listeners
	const cleanupFunctions: (() => void)[] = [];

	if ("serviceWorker" in navigator) {
		// Service worker message listener
		navigator.serviceWorker.addEventListener("message", handleSWMessage);
		cleanupFunctions.push(() => {
			navigator.serviceWorker.removeEventListener("message", handleSWMessage);
		});

		// Controller change listener
		const handleControllerChange = () => {
			debugLog("[App] Service worker controller changed");
		};
		navigator.serviceWorker.addEventListener("controllerchange", handleControllerChange);
		cleanupFunctions.push(() => {
			navigator.serviceWorker.removeEventListener("controllerchange", handleControllerChange);
		});
	}

	// Visibility change listener
	document.addEventListener("visibilitychange", handleVisibilityChange);
	cleanupFunctions.push(() => {
		document.removeEventListener("visibilitychange", handleVisibilityChange);
	});

	// Return cleanup function
	return () => {
		cleanupFunctions.forEach((cleanup) => cleanup());
	};
}
