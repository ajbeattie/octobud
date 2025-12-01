<script lang="ts">
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

	// ============================================================================
	// IMPORTS
	// ============================================================================

	// CSS
	import "../app.css";

	// Framework & Core
	import { getContext, onDestroy, onMount, setContext } from "svelte";
	import { get } from "svelte/store";
	import { browser } from "$app/environment";
	import { goto, afterNavigate } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { page as pageStore } from "$app/stores";
	import { invalidateAll } from "$app/navigation";
	import type { NavigationState } from "$lib/state/interfaces/common";
	import type { LayoutData } from "./$types";
	import { isAuthenticated, getAuthToken } from "$lib/stores/authStore";

	// Components - Dialogs
	import ViewDialog from "$lib/components/dialogs/ViewDialog.svelte";
	import ConfirmDialog from "$lib/components/dialogs/ConfirmDialog.svelte";
	import BulkActionConfirmDialog from "$lib/components/dialogs/BulkActionConfirmDialog.svelte";
	import SyntaxGuideDialog from "$lib/components/dialogs/SyntaxGuideDialog.svelte";
	import CustomSnoozeDateDialog from "$lib/components/dialogs/CustomSnoozeDateDialog.svelte";
	import CredentialsModal from "$lib/components/dialogs/CredentialsModal.svelte";

	// Components - Layout
	import PageSidebar from "$lib/components/sidebar/PageSidebar.svelte";
	import PageHeader from "$lib/components/shared/PageHeader.svelte";
	import ToastContainer from "$lib/components/shared/ToastContainer.svelte";
	import LayoutSkeleton from "$lib/components/shared/LayoutSkeleton.svelte";

	// State Controllers
	import { createViewDialogController } from "$lib/state/viewDialogController";
	import { createNotificationPageController } from "$lib/state/notificationPageController";

	// Stores
	import { toastStore } from "$lib/stores/toastStore";

	// API & Types
	import { fetchViews } from "$lib/api/views";
	import type { Tag } from "$lib/api/tags";

	// Stores
	import { currentTime } from "$lib/stores/timeStore";

	// Utilities
	import {
		registerServiceWorker,
		requestNotificationPermission,
		getNotificationPermission,
		sendMessageToSW,
		sendNotificationSettingToSW,
	} from "$lib/utils/serviceWorkerRegistration";
	import {
		getNotificationSettingsStore,
		isNotificationEnabled,
	} from "$lib/stores/notificationSettings";
	import { getThemeStore } from "$lib/stores/themeStore";
	import { setupServiceWorkerHandlers } from "$lib/utils/serviceWorkerMessageHandler";

	// Props
	export let data: LayoutData;

	// Component refs
	let pageSidebarComponent: any = null;
	let pageHeaderComponent: any = null;

	// ============================================================================
	// UNIFIED PAGE CONTROLLER - SINGLE SOURCE OF TRUTH
	// ============================================================================

	// Store current tags for getTags function
	let currentTags: Tag[] = data.tags ?? [];

	// Update tags when data changes
	$: currentTags = data.tags ?? [];

	// ============================================================================
	// SERVICE WORKER MESSAGE LISTENER - Set up immediately, not in onMount
	// ============================================================================

	// Store reference to pageController for service worker handlers
	// This will be set after pageController is created
	let pageControllerRef: ReturnType<typeof createNotificationPageController> | null = null;

	// Set up all service worker handlers as early as possible
	// This ensures we don't miss messages sent from the service worker
	let swCleanup: (() => void) | null = null;
	if (browser) {
		swCleanup = setupServiceWorkerHandlers({
			getPageController: () => pageControllerRef,
		});
	}

	const pageController = createNotificationPageController(
		{
			views: data.views,
			selectedViewSlug: null,
			selectedViewId: null,
			baseFilters: [],
			initialPage: { items: [], total: 0, page: 1, pageSize: 50 },
			initialSearchTerm: "",
			initialQuickFilters: [],
			initialPageNumber: 1,
			initialQuery: "",
			viewQuery: "",
			apiError: null,
		} as any,
		{
			onRefresh: async () => {
				await invalidateAll();
			},
			onRefreshViewCounts: async () => {
				await invalidateAll();
			},
			navigateToUrl: async (
				url: string,
				navOptions?: import("$lib/state/interfaces/common").NavigateOptions
			) => {
				// Extract pathname and search from URL
				const urlObj = new URL(
					url,
					typeof window !== "undefined" ? window.location.origin : "http://localhost"
				);
				await goto(resolve(urlObj.pathname as any) + urlObj.search, {
					replaceState: navOptions?.replace ?? true, // Default to replace for backwards compat
					noScroll: true,
					keepFocus: true,
					state: navOptions?.state,
				});
			},
			requestBulkConfirmation: async (action: string, count: number) => {
				return new Promise((resolve) => {
					bulkConfirmResolve = resolve;
					bulkConfirmDialogOpen = true;
					bulkConfirmAction = action;
					bulkConfirmCount = count;
				});
			},
			getTags: () => currentTags,
		}
	);

	// Set context immediately
	setContext("notificationPageController", pageController);

	// Update the service worker message handler with the pageController reference
	pageControllerRef = pageController;

	// Expose view dialog actions via context for child pages
	setContext("viewDialogActions", {
		startEditingWithQuery: null, // Will be set after viewDialogController is created
		openNewDialogWithQuery: null,
	});

	// Expose layout functions via context for child pages
	setContext("layoutFunctions", {
		showMoreShortcuts: null, // Will be set below
		showQueryGuide: null,
		isAnyDialogOpen: null,
		isShortcutsModalOpen: null,
		toggleShortcutsModal: null,
		getCommandPalette: () => pageHeaderComponent?.getCommandPalette(),
	});

	// ============================================================================
	// EXTRACT STORES FROM PAGE CONTROLLER
	// ============================================================================

	const {
		views,
		selectedViewId,
		quickQuery,
		hydrated,
		sidebarCollapsed,
		customDateDialogOpen,
		bulkUpdating,
	} = pageController.stores;

	const { builtInViewList, selectedViewSlug, defaultViewSlug, defaultViewDisplayName } =
		pageController.derived;

	// ============================================================================
	// BULK ACTION CONFIRMATION DIALOG STATE
	// ============================================================================

	let bulkConfirmDialogOpen = false;
	let bulkConfirmAction = "";
	let bulkConfirmCount = 0;
	let bulkConfirmResolve: ((confirmed: boolean) => void) | null = null;

	// ============================================================================
	// VIEW DIALOG CONTROLLER
	// ============================================================================

	async function refreshViewCounts() {
		if (typeof window === "undefined") {
			return;
		}

		try {
			const updatedViews = await fetchViews();
			pageController.actions.setViews(updatedViews);
		} catch (error) {
			console.error("Failed to refresh view counts:", error);
		}
	}

	async function selectViewBySlug(slug: string, shouldInvalidate: boolean = false) {
		await pageController.actions.selectViewBySlug(slug, shouldInvalidate);
	}

	const {
		stores: {
			open: viewDialogOpen,
			saving: viewDialogSaving,
			editing: editingView,
			confirmDeleteOpen,
			linkedRulesConfirmOpen,
			linkedRuleCount,
			draft: viewDraft,
			error: viewDialogError,
		},
		actions: {
			openNewDialog,
			openNewDialogWithQuery,
			startEditing,
			startEditingWithQuery,
			closeDialog: closeViewDialog,
			handleSave: handleViewSave,
			clearError: clearViewError,
			requestDelete: requestViewDelete,
			cancelDelete: cancelViewDelete,
			confirmDelete: confirmDeleteView,
			confirmLinkedRulesDelete,
			cancelLinkedRulesDelete,
		},
	} = createViewDialogController({
		views,
		setViews: pageController.actions.setViews,
		selectedViewId,
		invalidateViews: refreshViewCounts,
		navigateToSlug: selectViewBySlug,
		refreshNotifications: pageController.actions.refresh,
		quickQuery,
	});

	// ============================================================================
	// UPDATE CONTEXT WITH VIEW DIALOG ACTIONS
	// ============================================================================

	const viewDialogActionsContext = getContext("viewDialogActions") as any;
	viewDialogActionsContext.startEditingWithQuery = startEditingWithQuery;
	viewDialogActionsContext.openNewDialogWithQuery = openNewDialogWithQuery;

	// ============================================================================
	// OTHER DIALOG STATE
	// ============================================================================

	let syntaxGuideOpen = false;
	let credentialsModalOpen = false;

	// ============================================================================
	// UPDATE CONTEXT WITH LAYOUT FUNCTIONS
	// ============================================================================

	const layoutFunctionsContext = getContext("layoutFunctions") as any;
	layoutFunctionsContext.showMoreShortcuts = handleShowMoreShortcuts;
	layoutFunctionsContext.showQueryGuide = () => {
		syntaxGuideOpen = true;
	};
	layoutFunctionsContext.isAnyDialogOpen = isAnyDialogOpen;
	layoutFunctionsContext.isShortcutsModalOpen = isShortcutsModalOpen;
	layoutFunctionsContext.toggleShortcutsModal = toggleShortcutsModal;

	// ============================================================================
	// HANDLER FUNCTIONS
	// ============================================================================

	async function handleLogoClick() {
		// Always navigate to inbox without query params
		if (typeof window === "undefined") return;

		const url = new URL(window.location.href);
		url.pathname = `/views/inbox`;
		url.search = "";

		await goto(resolve(url.pathname as any) + url.search, {
			keepFocus: true,
			noScroll: true,
			replaceState: false,
			invalidateAll: false,
		});
	}

	function handleCustomSnoozeConfirm(until: string) {
		const notificationId = get(pageController.stores.customDateDialogNotificationId);
		if (!notificationId) return;

		const pageData = get(pageController.stores.pageData);
		const notification = pageData.items.find((n) => (n.githubId ?? n.id) === notificationId);
		if (notification) {
			void pageController.actions.snooze(notification, until);
		}
		pageController.actions.closeCustomSnoozeDialog();
	}

	function handleBulkConfirm() {
		if (bulkConfirmResolve) {
			bulkConfirmResolve(true);
			bulkConfirmResolve = null;
		}
		bulkConfirmDialogOpen = false;
	}

	function handleBulkCancel() {
		if (bulkConfirmResolve) {
			bulkConfirmResolve(false);
			bulkConfirmResolve = null;
		}
		bulkConfirmDialogOpen = false;
	}

	function handleShowMoreShortcuts() {
		pageSidebarComponent?.openShortcutsModal();
	}

	function handleShowQueryGuide() {
		syntaxGuideOpen = true;
	}

	function handleOpenCredentialsModal() {
		credentialsModalOpen = true;
	}

	function handleCloseCredentialsModal() {
		credentialsModalOpen = false;
	}

	function toggleShortcutsModal(): boolean {
		pageSidebarComponent?.toggleShortcutsModal();
		return true;
	}

	function isShortcutsModalOpen(): boolean {
		return pageSidebarComponent?.isShortcutsModalOpen() ?? false;
	}

	function isAnyDialogOpen(): boolean {
		const commandPalette = pageHeaderComponent?.getCommandPalette();
		return (
			$viewDialogOpen ||
			$confirmDeleteOpen ||
			$linkedRulesConfirmOpen ||
			bulkConfirmDialogOpen ||
			$customDateDialogOpen ||
			isShortcutsModalOpen() ||
			commandPalette?.isPaletteOpen() === true
		);
	}

	// ============================================================================
	// KEYBOARD SHORTCUTS - Will be scoped based on route
	// ============================================================================

	let unregisterShortcuts: (() => void) | null = null;

	// Initialize time store to keep timestamps refreshed
	// Subscribe to currentTime store to ensure it stays active
	$: _timeStoreValue = $currentTime;

	onMount(() => {
		// Hide splash screen once component is mounted
		if (browser) {
			document.body.classList.remove("loading");

			// Check authentication and redirect if needed
			const token = getAuthToken();
			const isLoginPage = window.location.pathname === "/login";

			if (!token && !isLoginPage) {
				void goto(resolve("/login" as any));
				return; // Don't initialize app features if redirecting to login
			} else if (token && isLoginPage) {
				void goto(resolve("/views/inbox" as any));
				return; // Don't initialize app features if redirecting away from login
			}

			// Only initialize app features if not on login page
			if (!isLoginPage) {
				void refreshViewCounts();
			}
		}

		// Initialize theme store (only if not on login page)
		if (browser && window.location.pathname !== "/login") {
			getThemeStore();
		}

		// Register service worker for background sync (only if not on login page)
		let controllerChangeHandler: (() => Promise<void>) | null = null;
		if (browser && window.location.pathname !== "/login") {
			const notificationSettings = getNotificationSettingsStore();

			const sendSettingsToSW = async () => {
				const permission = getNotificationPermission();
				const enabled = isNotificationEnabled();
				await sendMessageToSW({
					type: "NOTIFICATION_PERMISSION",
					permission,
				});
				await sendNotificationSettingToSW(enabled);
			};

			void registerServiceWorker().then(async (registration) => {
				if (registration) {
					// Send current settings to SW
					if (navigator.serviceWorker.controller) {
						await sendSettingsToSW();
					} else {
						// Wait for controller to be available
						controllerChangeHandler = async () => {
							await sendSettingsToSW();
							if (controllerChangeHandler) {
								navigator.serviceWorker.removeEventListener(
									"controllerchange",
									controllerChangeHandler
								);
								controllerChangeHandler = null;
							}
						};
						navigator.serviceWorker.addEventListener("controllerchange", controllerChangeHandler);
					}

					// Send initial notification setting to SW
					const enabled = isNotificationEnabled();
					await sendNotificationSettingToSW(enabled);

					// Request permission if notifications are enabled and permission not granted
					if (enabled) {
						await requestNotificationPermission();
					}
				}
			});
		}

		// Notification permission is handled by service worker registration
		// Keyboard shortcuts are registered per-route (see views/[slug]/+page.svelte, settings/+page.svelte)

		return () => {
			if (unregisterShortcuts) {
				unregisterShortcuts();
			}
			// Clean up service worker event listener if it was added
			if (browser && controllerChangeHandler && "serviceWorker" in navigator) {
				navigator.serviceWorker.removeEventListener("controllerchange", controllerChangeHandler);
				controllerChangeHandler = null;
			}
		};
	});

	onDestroy(() => {
		// Clean up service worker message listener
		if (swCleanup) {
			swCleanup();
		}

		// Only destroy page controller if it was initialized (not on login page)
		if (!isLoginRoute) {
			pageController.actions.destroy();
		}
		// Clean up timeStore interval
		if (currentTime && typeof currentTime.destroy === "function") {
			currentTime.destroy();
		}
	});

	// Restore UI state from browser history when navigating back/forward
	afterNavigate(({ from, to, type }) => {
		// Only restore state on browser back/forward (popstate) navigation
		if (type !== "popstate") return;

		const navState = $pageStore.state as NavigationState | undefined;
		if (!navState) return;

		// Restore sidebar state
		if (navState.sidebarCollapsed !== undefined) {
			sidebarCollapsed.set(navState.sidebarCollapsed);
		}

		// Restore split mode state
		if (navState.splitModeEnabled !== undefined) {
			pageController.stores.splitModeEnabled.set(navState.splitModeEnabled);
		}

		// Restore scroll position (for SingleMode list view)
		if (navState.savedScrollPosition !== undefined) {
			pageController.stores.savedListScrollPosition.set(navState.savedScrollPosition);
		}

		// Restore keyboard focus index
		if (navState.keyboardFocusIndex !== undefined) {
			pageController.stores.keyboardFocusIndex.set(navState.keyboardFocusIndex);
		}
	});

	// ============================================================================
	// REACTIVE: Update views when data changes
	// ============================================================================

	$: if (!isLoginRoute) {
		pageController.actions.setViews(data.views);
	}

	// ============================================================================
	// COMPUTED VALUES FOR SIDEBAR
	// ============================================================================

	$: systemViewsBySlug = new Map($views.filter((v) => v.systemView).map((v) => [v.slug, v]));

	$: inboxView = systemViewsBySlug.get("inbox");

	// ============================================================================
	// DOCUMENT TITLE
	// ============================================================================

	$: pageTitle = isLoginRoute
		? "Login - Octobud"
		: (() => {
				const unreadCount = inboxView?.unreadCount ?? 0;
				return unreadCount > 0 ? `Octobud (${unreadCount})` : "Octobud";
			})();

	// ============================================================================
	// ROUTE-SPECIFIC LOGIC
	// ============================================================================

	$: isSettingsRoute = $pageStore.url.pathname.startsWith("/settings");
	$: isViewsRoute =
		$pageStore.url.pathname.startsWith("/views/") || $pageStore.url.pathname === "/";
	$: isLoginRoute = $pageStore.url.pathname === "/login";

	// Close detail modal when navigating to settings
	$: if (!isLoginRoute && isSettingsRoute) {
		const detailOpen = get(pageController.stores.detailOpen);
		if (detailOpen) {
			pageController.actions.closeDetail();
		}
	}
</script>

<svelte:head>
	<title>{pageTitle}</title>
</svelte:head>

{#if $hydrated}
	{#if isLoginRoute}
		<!-- Login page uses its own layout - just render slot -->
		<slot />
	{:else}
		<div class="h-screen flex flex-col bg-white dark:bg-gray-950">
			<!-- Full-width header at top - always use PageHeader -->
			<PageHeader
				bind:this={pageHeaderComponent}
				defaultViewSlug={$defaultViewSlug}
				defaultViewDisplayName={$defaultViewDisplayName}
				detailOpen={false}
				sidebarCollapsed={$sidebarCollapsed}
				onToggleSidebar={pageController.actions.toggleSidebar}
				builtInViews={$builtInViewList}
				views={$views}
				tags={data.tags}
				selectedViewSlug={$selectedViewSlug}
				onLogoClick={handleLogoClick}
				onSelectView={selectViewBySlug}
				onShowMoreShortcuts={handleShowMoreShortcuts}
				onShowQueryGuide={handleShowQueryGuide}
				onOpenCredentialsModal={handleOpenCredentialsModal}
			/>

			<!-- Sidebar and content area below header -->
			<div class="flex-1 flex min-h-0 overflow-hidden">
				<!-- Sidebar -->
				<PageSidebar
					bind:this={pageSidebarComponent}
					builtInViewList={$builtInViewList}
					views={$views}
					tags={data.tags}
					selectedViewId={$selectedViewId}
					selectedViewSlug={$selectedViewSlug}
					{inboxView}
					collapsed={$sidebarCollapsed}
					onNewView={openNewDialog}
					onEditView={startEditing}
				/>

				<!-- Main content area -->
				<div class="flex-1 flex flex-col overflow-hidden">
					<!-- Child routes render in slot -->
					<slot />
				</div>
			</div>
		</div>
	{/if}
{:else if isLoginRoute}
	<!-- Login page - no skeleton needed -->
	<slot />
{:else}
	<LayoutSkeleton />
{/if}

{#if !isLoginRoute}
	<!-- Dialogs and modals - only show when not on login page -->
	<ViewDialog
		open={$viewDialogOpen}
		saving={$viewDialogSaving}
		initialValue={$viewDraft}
		error={$viewDialogError}
		onDelete={$editingView ? requestViewDelete : null}
		onClose={closeViewDialog}
		onSave={handleViewSave}
		onClearError={clearViewError}
	/>

	<ConfirmDialog
		open={$confirmDeleteOpen}
		title="Delete view"
		body="Are you sure you want to delete this view? This action cannot be undone."
		confirmLabel="Delete"
		cancelLabel="Cancel"
		confirmTone="danger"
		confirming={$viewDialogSaving}
		onCancel={cancelViewDelete}
		onConfirm={confirmDeleteView}
	/>

	<ConfirmDialog
		open={$linkedRulesConfirmOpen}
		title="Delete view with linked rules"
		body="This view has {$linkedRuleCount} linked rule{$linkedRuleCount === 1
			? ''
			: 's'}. Deleting the view will also delete {$linkedRuleCount === 1
			? 'this rule'
			: 'these rules'}. Continue?"
		confirmLabel="Delete view and rules"
		cancelLabel="Cancel"
		confirmTone="danger"
		confirming={$viewDialogSaving}
		onCancel={cancelLinkedRulesDelete}
		onConfirm={confirmLinkedRulesDelete}
	/>

	<CustomSnoozeDateDialog
		isOpen={$customDateDialogOpen}
		onConfirm={handleCustomSnoozeConfirm}
		onClose={pageController.actions.closeCustomSnoozeDialog}
	/>

	<BulkActionConfirmDialog
		open={bulkConfirmDialogOpen}
		action={bulkConfirmAction}
		count={bulkConfirmCount}
		onConfirm={handleBulkConfirm}
		onCancel={handleBulkCancel}
		confirming={$bulkUpdating}
	/>

	<SyntaxGuideDialog bind:open={syntaxGuideOpen} onClose={() => (syntaxGuideOpen = false)} />

	<CredentialsModal open={credentialsModalOpen} onClose={handleCloseCredentialsModal} />

	<ToastContainer />
{/if}
