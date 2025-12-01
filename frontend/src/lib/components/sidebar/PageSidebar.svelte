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

	import { getContext } from "svelte";
	import { page as pageStore } from "$app/stores";
	import { goto } from "$app/navigation";
	import { resolve } from "$app/paths";
	import { invalidateAll } from "$app/navigation";
	import SidebarCollapsed from "./SidebarCollapsed.svelte";
	import SidebarExpanded from "./SidebarExpanded.svelte";
	import SidebarHover from "./SidebarHover.svelte";
	import ShortcutsModal from "../dialogs/ShortcutsModal.svelte";
	import ViewDialog from "../dialogs/ViewDialog.svelte";
	import TagDialog from "../dialogs/TagDialog.svelte";
	import ConfirmDialog from "../dialogs/ConfirmDialog.svelte";
	import type { NotificationView } from "$lib/api/types";
	import type { NotificationPageController } from "$lib/state/types";
	import { reorderViews } from "$lib/api/views";
	import type { Tag } from "$lib/api/tags";
	import { fetchTags, reorderTags, deleteTag } from "$lib/api/tags";
	import { toastStore } from "$lib/stores/toastStore";

	// Get page controller from context
	const pageController = getContext<NotificationPageController>("notificationPageController");

	export let builtInViewList: NotificationView[];
	export let views: NotificationView[];
	export let tags: Tag[];
	export let selectedViewId: string;
	export let selectedViewSlug: string;
	export let inboxView: NotificationView | undefined = undefined;
	export let onNewView: () => void;
	export let onEditView: (view: NotificationView) => void;
	export let collapsed: boolean = false;

	let showShortcutsModal = false;
	let viewDialogOpen = false;
	let selectedViewForEdit: NotificationView | null = null;
	let tagDialogOpen = false;
	let selectedTagForEdit: Tag | null = null;
	let tagDeleteConfirmOpen = false;
	let tagDeleting = false;
	let reorderMode = false;
	let draggedViewId: string | null = null;
	let draggedOverViewId: string | null = null;
	let localViewOrder: NotificationView[] = [];
	let originalViewOrder: NotificationView[] = [];
	let reordering = false;
	let hoverExpanded = false;
	let hoverTimeout: ReturnType<typeof setTimeout> | null = null;

	// Views collapsed state - initialize from localStorage
	const VIEWS_COLLAPSED_KEY = "octobud:sidebar:viewsCollapsed";
	const storedViewsCollapsed =
		typeof window !== "undefined" ? localStorage.getItem(VIEWS_COLLAPSED_KEY) === "true" : false;
	let viewsCollapsed = storedViewsCollapsed;

	// Tags collapsed state - initialize from localStorage
	const TAGS_COLLAPSED_KEY = "octobud:sidebar:tagsCollapsed";
	const storedTagsCollapsed =
		typeof window !== "undefined" ? localStorage.getItem(TAGS_COLLAPSED_KEY) === "true" : false;
	let tagsCollapsed = storedTagsCollapsed;
	let tagReorderMode = false;
	let draggedTagId: string | null = null;
	let draggedOverTagId: string | null = null;
	let localTagOrder: Tag[] = [];
	let originalTagOrder: Tag[] = [];
	let tagReordering = false;

	// Derive the currently selected tag from URL
	$: currentTagSlug = (() => {
		const currentPath = $pageStore.url.pathname;
		const tagViewMatch = currentPath.match(/^\/views\/tag-(.+)$/);
		return tagViewMatch ? decodeURIComponent(tagViewMatch[1]) : null;
	})();

	$: selectedTagId = currentTagSlug ? (tags.find((t) => t.slug === currentTagSlug)?.id ?? "") : "";

	// Determine if we're on a tag view (to avoid highlighting views)
	$: isTagView = currentTagSlug !== null;

	// Only use view selection if not on a tag view
	$: effectiveSelectedViewId = isTagView ? "" : selectedViewId;
	$: effectiveSelectedViewSlug = isTagView ? "" : selectedViewSlug;

	function handleShowShortcuts() {
		showShortcutsModal = true;
	}

	function handleCloseShortcuts() {
		showShortcutsModal = false;
	}

	// Export function to allow external components to open the modal
	export function openShortcutsModal() {
		showShortcutsModal = true;
	}

	// Export function to toggle the modal
	export function toggleShortcutsModal() {
		showShortcutsModal = !showShortcutsModal;
	}

	// Export function to check if modal is open
	export function isShortcutsModalOpen() {
		return showShortcutsModal;
	}

	$: userViews = views.filter((v) => !v.systemView);

	// Check if we're on settings route
	$: isSettingsRoute = $pageStore.url.pathname.startsWith("/settings");

	// Sort views by display order or alphabetically
	$: sortedUserViews = (() => {
		// In reorder mode, use the local order as-is (user is manually ordering)
		if (reorderMode) {
			return localViewOrder;
		}

		// Otherwise, sort by display order or alphabetically
		// Note: displayOrder can be 0 or any number, null/undefined means no order set
		const allHaveOrder = userViews.every((v) => v.displayOrder != null);

		if (allHaveOrder) {
			return [...userViews].sort((a, b) => (a.displayOrder || 0) - (b.displayOrder || 0));
		} else {
			return [...userViews].sort((a, b) => a.name.localeCompare(b.name));
		}
	})();

	// Sort tags by display order (server returns them sorted)
	$: sortedTags = (() => {
		// In reorder mode, use the local order as-is (user is manually ordering)
		if (tagReorderMode) {
			return localTagOrder;
		}

		// Otherwise, use the order from the server (sorted by display_order, then name)
		return tags;
	})();

	function toggleReorderMode() {
		if (reorderMode) {
			// Exit reorder mode - save changes
			void handleSaveReorder();
		} else {
			// Enter reorder mode
			reorderMode = true;
			originalViewOrder = [...sortedUserViews];
			localViewOrder = [...sortedUserViews];
		}
	}

	async function handleSaveReorder() {
		reordering = true;
		try {
			const viewIds = localViewOrder.map((v) => v.id);
			await reorderViews(viewIds);

			// Refresh views to get updated order and unread counts
			// This uses fetchViews() which calls ListViewsWithCounts and includes unread counts
			await pageController.actions.refreshViewCounts();
		} catch (error) {
			console.error("Failed to reorder views:", error);
			// Revert to original order on error
			localViewOrder = [...originalViewOrder];
		} finally {
			reorderMode = false;
			reordering = false;
			draggedViewId = null;
			draggedOverViewId = null;
		}
	}

	function handleDragStart(viewId: string) {
		draggedViewId = viewId;
	}

	function handleDragOver(e: DragEvent, targetViewId: string) {
		if (!draggedViewId || draggedViewId === targetViewId) {
			draggedOverViewId = null;
			return;
		}

		draggedOverViewId = targetViewId;
	}

	function handleDrop(e: DragEvent, targetViewId: string) {
		if (!draggedViewId || draggedViewId === targetViewId) {
			draggedOverViewId = null;
			return;
		}

		// Reorder the local list
		const draggedIndex = localViewOrder.findIndex((v) => v.id === draggedViewId);
		const targetIndex = localViewOrder.findIndex((v) => v.id === targetViewId);

		if (draggedIndex !== -1 && targetIndex !== -1) {
			const newOrder = [...localViewOrder];
			const [draggedItem] = newOrder.splice(draggedIndex, 1);
			newOrder.splice(targetIndex, 0, draggedItem);
			localViewOrder = newOrder;
		}

		draggedOverViewId = null;
	}

	function handleDragEnd() {
		draggedViewId = null;
		draggedOverViewId = null;
	}

	// Tag handlers
	function handleSelectTag(tagId: string) {
		// Find the tag by ID and navigate to tag view using special slug format
		const tag = tags.find((t) => t.id === tagId);
		if (tag) {
			// Navigate to tag as a special view: /views/tag-{slug}
			const route = `/views/tag-${encodeURIComponent(tag.slug)}`;
			void goto(resolve(route as any));
		}
	}

	function handleEditTag(tag: Tag) {
		selectedTagForEdit = tag;
		tagDialogOpen = true;
	}

	function handleNewTag() {
		selectedTagForEdit = null;
		tagDialogOpen = true;
	}

	async function handleTagDialogClose() {
		// Store the tag we were editing before closing
		const editedTagId = selectedTagForEdit?.id;

		// Refresh tags list
		try {
			const newTags = await fetchTags();
			const oldTags = tags;
			tags = newTags;

			// Check if we're currently viewing a tag
			if (currentTagSlug) {
				// Check if the tag we were viewing still exists
				const stillExists = newTags.some((t) => t.slug === currentTagSlug);

				if (!stillExists) {
					// Tag was deleted, navigate to inbox
					await goto(resolve("/views/inbox" as any));
					await invalidateAll();
				} else if (editedTagId) {
					// Check if the edited tag's slug changed (name changed)
					const updatedTag = newTags.find((t) => t.id === editedTagId);
					const oldTag = oldTags.find((t) => t.id === editedTagId);

					if (
						updatedTag &&
						oldTag &&
						updatedTag.slug !== oldTag.slug &&
						oldTag.slug === currentTagSlug
					) {
						// We were viewing this tag and its slug changed, navigate to new slug
						const route = `/views/tag-${updatedTag.slug}`;
						await goto(resolve(route as any));
					}
					// Invalidate to refresh the current view
					await invalidateAll();
				} else {
					// Just refresh the current view (tag metadata might have changed)
					await invalidateAll();
				}
			}
			// If not on a tag view, just update the tags list (no navigation needed)
		} catch (error) {
			console.error("Failed to refresh tags:", error);
		}

		// Close dialog and clear state
		tagDialogOpen = false;
		selectedTagForEdit = null;
	}

	function handleToggleTagsCollapsed() {
		tagsCollapsed = !tagsCollapsed;
		if (typeof window !== "undefined") {
			localStorage.setItem(TAGS_COLLAPSED_KEY, tagsCollapsed.toString());
		}
	}

	function toggleTagReorderMode() {
		if (tagReorderMode) {
			// Exit reorder mode - save changes
			void handleSaveTagReorder();
		} else {
			// Enter reorder mode
			tagReorderMode = true;
			originalTagOrder = [...sortedTags];
			localTagOrder = [...sortedTags];
		}
	}

	async function handleSaveTagReorder() {
		tagReordering = true;
		try {
			const tagIds = localTagOrder.map((t) => t.id);
			const updatedTags = await reorderTags(tagIds);
			tags = updatedTags;
		} catch (error) {
			console.error("Failed to reorder tags:", error);
			// Revert to original order on error
			localTagOrder = [...originalTagOrder];
		} finally {
			tagReorderMode = false;
			tagReordering = false;
			draggedTagId = null;
			draggedOverTagId = null;
		}
	}

	function handleTagDragStart(tagId: string) {
		draggedTagId = tagId;
	}

	function handleTagDragOver(e: DragEvent, targetTagId: string) {
		if (!draggedTagId || draggedTagId === targetTagId) {
			draggedOverTagId = null;
			return;
		}

		draggedOverTagId = targetTagId;
	}

	function handleTagDrop(e: DragEvent, targetTagId: string) {
		if (!draggedTagId || draggedTagId === targetTagId) {
			draggedOverTagId = null;
			return;
		}

		// Reorder the local list
		const draggedIndex = localTagOrder.findIndex((t) => t.id === draggedTagId);
		const targetIndex = localTagOrder.findIndex((t) => t.id === targetTagId);

		if (draggedIndex !== -1 && targetIndex !== -1) {
			const newOrder = [...localTagOrder];
			const [draggedItem] = newOrder.splice(draggedIndex, 1);
			newOrder.splice(targetIndex, 0, draggedItem);
			localTagOrder = newOrder;
		}

		draggedOverTagId = null;
	}

	function handleTagDragEnd() {
		draggedTagId = null;
		draggedOverTagId = null;
	}

	function handleMouseEnter() {
		if (!collapsed) return;

		// Add a small delay before expanding to prevent accidental triggers
		hoverTimeout = setTimeout(() => {
			hoverExpanded = true;
		}, 200);
	}

	function handleMouseLeave() {
		if (hoverTimeout) {
			clearTimeout(hoverTimeout);
			hoverTimeout = null;
		}
		hoverExpanded = false;
	}

	// Separate handler for the overlay to keep it open
	function handleOverlayMouseEnter() {
		// Clear any pending timeout and keep expanded
		if (hoverTimeout) {
			clearTimeout(hoverTimeout);
			hoverTimeout = null;
		}
		hoverExpanded = true;
	}

	function handleOverlayMouseLeave() {
		hoverExpanded = false;
	}

	function handleSelectView(slug: string) {
		pageController.actions.selectViewBySlug(slug);
	}

	function handleToggleViewsCollapsed() {
		viewsCollapsed = !viewsCollapsed;
		if (typeof window !== "undefined") {
			localStorage.setItem(VIEWS_COLLAPSED_KEY, viewsCollapsed.toString());
		}
	}

	// Tag delete handlers
	function requestTagDelete() {
		tagDeleteConfirmOpen = true;
	}

	function cancelTagDelete() {
		tagDeleteConfirmOpen = false;
	}

	async function confirmTagDelete() {
		if (!selectedTagForEdit) return;

		tagDeleting = true;
		try {
			await deleteTag(selectedTagForEdit.id);
			toastStore.show("Tag deleted successfully", "success");

			// Close dialogs
			tagDeleteConfirmOpen = false;
			tagDialogOpen = false;

			// Refresh tags and handle navigation
			await handleTagDialogClose();
		} catch (error) {
			toastStore.show(`Failed to delete tag: ${error}`, "error");
		} finally {
			tagDeleting = false;
		}
	}
</script>

<!-- Container for both collapsed sidebar and hover overlay -->
<div class="relative flex-shrink-0 h-screen">
	<aside
		class="sticky top-0 flex h-full max-h-screen bg-gray-100 dark:bg-gray-950 {collapsed
			? 'w-13'
			: 'w-[15rem] xl:w-[16rem] 2xl:w-[18rem]'} {collapsed
			? ' pr-0'
			: 'pl-2'} flex-col overflow-hidden border-r border-gray-200 dark:border-gray-800 transition-all duration-200"
		on:mouseenter={handleMouseEnter}
		on:mouseleave={handleMouseLeave}
	>
		{#if collapsed}
			<SidebarCollapsed
				{builtInViewList}
				{sortedUserViews}
				selectedViewId={effectiveSelectedViewId}
				selectedViewSlug={effectiveSelectedViewSlug}
				{inboxView}
				{isSettingsRoute}
				{viewsCollapsed}
				{sortedTags}
				{selectedTagId}
				{tagsCollapsed}
				onSelectView={handleSelectView}
				onToggleViewsCollapsed={handleToggleViewsCollapsed}
				onSelectTag={handleSelectTag}
				onToggleTagsCollapsed={handleToggleTagsCollapsed}
			/>
		{:else}
			<SidebarExpanded
				{builtInViewList}
				{sortedUserViews}
				{userViews}
				selectedViewId={effectiveSelectedViewId}
				selectedViewSlug={effectiveSelectedViewSlug}
				{inboxView}
				{isSettingsRoute}
				{viewsCollapsed}
				{reorderMode}
				{reordering}
				{draggedOverViewId}
				{tags}
				{sortedTags}
				{selectedTagId}
				{tagsCollapsed}
				{tagReorderMode}
				{tagReordering}
				{draggedOverTagId}
				onSelectView={handleSelectView}
				onToggleViewsCollapsed={handleToggleViewsCollapsed}
				{onNewView}
				{onEditView}
				onToggleReorderMode={toggleReorderMode}
				onDragStart={handleDragStart}
				onDragOver={handleDragOver}
				onDrop={handleDrop}
				onDragEnd={handleDragEnd}
				onSelectTag={handleSelectTag}
				onToggleTagsCollapsed={handleToggleTagsCollapsed}
				onNewTag={handleNewTag}
				onEditTag={handleEditTag}
				onToggleTagReorderMode={toggleTagReorderMode}
				onTagDragStart={handleTagDragStart}
				onTagDragOver={handleTagDragOver}
				onTagDrop={handleTagDrop}
				onTagDragEnd={handleTagDragEnd}
			/>
		{/if}
	</aside>

	<!-- Hover-expanded overlay - positioned outside the aside for better control -->
	{#if collapsed && hoverExpanded}
		<SidebarHover
			{builtInViewList}
			{sortedUserViews}
			{userViews}
			selectedViewId={effectiveSelectedViewId}
			selectedViewSlug={effectiveSelectedViewSlug}
			{inboxView}
			{isSettingsRoute}
			{viewsCollapsed}
			{reorderMode}
			{reordering}
			{draggedOverViewId}
			{tags}
			{sortedTags}
			{selectedTagId}
			{tagsCollapsed}
			{tagReorderMode}
			{tagReordering}
			{draggedOverTagId}
			onSelectView={handleSelectView}
			onToggleViewsCollapsed={handleToggleViewsCollapsed}
			{onNewView}
			{onEditView}
			onToggleReorderMode={toggleReorderMode}
			onDragStart={handleDragStart}
			onDragOver={handleDragOver}
			onDrop={handleDrop}
			onDragEnd={handleDragEnd}
			onSelectTag={handleSelectTag}
			onToggleTagsCollapsed={handleToggleTagsCollapsed}
			onNewTag={handleNewTag}
			onEditTag={handleEditTag}
			onToggleTagReorderMode={toggleTagReorderMode}
			onTagDragStart={handleTagDragStart}
			onTagDragOver={handleTagDragOver}
			onTagDrop={handleTagDrop}
			onTagDragEnd={handleTagDragEnd}
			onMouseEnter={handleOverlayMouseEnter}
			onMouseLeave={handleOverlayMouseLeave}
		/>
	{/if}
</div>

<ShortcutsModal open={showShortcutsModal} onClose={handleCloseShortcuts} />

<ViewDialog
	bind:open={viewDialogOpen}
	initialValue={selectedViewForEdit}
	on:close={() => {
		viewDialogOpen = false;
		selectedViewForEdit = null;
	}}
/>

<TagDialog
	bind:open={tagDialogOpen}
	tag={selectedTagForEdit}
	onDelete={selectedTagForEdit ? requestTagDelete : null}
	on:close={handleTagDialogClose}
/>

<ConfirmDialog
	open={tagDeleteConfirmOpen}
	title="Delete tag"
	body="Are you sure you want to delete this tag? This action cannot be undone."
	confirmLabel="Delete"
	cancelLabel="Cancel"
	confirmTone="danger"
	confirming={tagDeleting}
	onCancel={cancelTagDelete}
	onConfirm={confirmTagDelete}
/>
