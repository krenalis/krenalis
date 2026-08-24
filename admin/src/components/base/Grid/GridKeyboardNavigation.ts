import React from 'react';

const gridNavigationKeys = ['ArrowDown', 'ArrowLeft', 'ArrowRight', 'ArrowUp'] as const;
const gridKeyboardControlSelector = [
	'input',
	'select',
	'textarea',
	'[contenteditable]:not([contenteditable="false"])',
	'.draggable-wrapper__handle',
	'sl-input',
	'sl-menu',
	'sl-menu-item',
	'sl-option',
	'sl-radio',
	'sl-radio-button',
	'sl-radio-group',
	'sl-range',
	'sl-select',
	'sl-tab',
	'sl-tab-group',
	'sl-textarea',
	'sl-tree',
	'sl-tree-item',
	'[role="combobox"]',
	'[role="listbox"]',
	'[role="menu"]',
	'[role="menubar"]',
	'[role="menuitem"]',
	'[role="menuitemcheckbox"]',
	'[role="menuitemradio"]',
	'[role="option"]',
	'[role="radio"]',
	'[role="radiogroup"]',
	'[role="separator"]',
	'[role="slider"]',
	'[role="spinbutton"]',
	'[role="tab"]',
	'[role="tablist"]',
	'[role="textbox"]',
	'[role="tree"]',
	'[role="treegrid"]',
	'[role="treeitem"]',
].join(',');
const openKeyboardOverlaySelector = 'dialog[open], sl-dialog[open], sl-dropdown[open], sl-select[open]';

type GridNavigationKey = (typeof gridNavigationKeys)[number];

const focusGridForKeyboardNavigation = (event: React.MouseEvent<HTMLDivElement>) => {
	const target = event.target;
	if (!(target instanceof Element)) {
		return;
	}
	const row = target.closest('.grid__row--clickable');
	const expandButton = target.closest('.grid__row-expand');
	if ((row == null && expandButton == null) || target.closest('.grid') !== event.currentTarget) {
		return;
	}
	event.currentTarget.focus({ preventScroll: true });
};

const navigateGrid = (
	grid: HTMLElement,
	key: string,
	shiftKey: boolean,
	onMoveRow?: (overRowID: string, movedRowID: string) => void,
): boolean => {
	if (!isGridNavigationKey(key)) {
		return false;
	}
	const isMove = shiftKey && onMoveRow != null && (key === 'ArrowDown' || key === 'ArrowUp');
	if (shiftKey && !isMove) {
		return false;
	}
	const rows = Array.from(grid.querySelectorAll<HTMLElement>('.grid__row--clickable')).filter(
		(row) => row.closest('.grid') === grid && row.getClientRects().length > 0,
	);
	if (rows.length === 0) {
		return false;
	}

	const selectedIndex = rows.findIndex((row) => row.classList.contains('grid__row--selected'));
	const selectedRow = rows[selectedIndex];
	if (isMove) {
		if (selectedRow != null) {
			moveGridRow(selectedRow, key === 'ArrowUp' ? -1 : 1, onMoveRow);
		}
		return true;
	}
	if (key === 'ArrowDown') {
		selectGridRow(rows[selectedIndex === -1 ? 0 : Math.min(selectedIndex + 1, rows.length - 1)]);
		return true;
	}
	if (key === 'ArrowUp') {
		selectGridRow(rows[selectedIndex === -1 ? rows.length - 1 : Math.max(selectedIndex - 1, 0)]);
		return true;
	}
	if (selectedRow == null) {
		return true;
	}

	const nestedRows = selectedRow.closest<HTMLElement>('.grid__nested-rows');
	if (key === 'ArrowRight') {
		if (!selectedRow.classList.contains('grid__row--parent') || nestedRows == null) {
			return true;
		}
		if (!nestedRows.classList.contains('grid__nested-rows--expanded')) {
			getDirectExpandButton(nestedRows)?.click();
			return true;
		}
		const nextRow = rows[selectedIndex + 1];
		if (nextRow != null && nestedRows.contains(nextRow)) {
			selectGridRow(nextRow);
		}
		return true;
	}

	if (key === 'ArrowLeft') {
		if (
			selectedRow.classList.contains('grid__row--parent') &&
			nestedRows?.classList.contains('grid__nested-rows--expanded')
		) {
			getDirectExpandButton(nestedRows)?.click();
			return true;
		}
		let parentRows = nestedRows;
		if (selectedRow.classList.contains('grid__row--parent')) {
			parentRows = nestedRows?.parentElement?.closest<HTMLElement>('.grid__nested-rows') || null;
		}
		const parentRow = parentRows == null ? null : getDirectParentRow(parentRows);
		if (parentRow != null) {
			selectGridRow(parentRow);
		}
		return true;
	}
	return false;
};

const navigateGridWithKeyboard = (
	event: React.KeyboardEvent<HTMLDivElement>,
	onMoveRow?: (overRowID: string, movedRowID: string) => void,
) => {
	if (event.target !== event.currentTarget || event.altKey || event.ctrlKey || event.metaKey) {
		return;
	}
	if (navigateGrid(event.currentTarget, event.key, event.shiftKey, onMoveRow)) {
		event.preventDefault();
	}
};

const shouldNavigateGridFromDocument = (event: KeyboardEvent): boolean => {
	if (
		event.defaultPrevented ||
		event.isComposing ||
		event.altKey ||
		event.ctrlKey ||
		event.metaKey ||
		!isGridNavigationKey(event.key) ||
		document.querySelector(openKeyboardOverlaySelector) != null
	) {
		return false;
	}
	for (const target of event.composedPath()) {
		if (target instanceof Element && target.matches(gridKeyboardControlSelector)) {
			return false;
		}
	}
	return true;
};

const getDirectExpandButton = (nestedRows: HTMLElement): HTMLElement | null => {
	for (const child of Array.from(nestedRows.children)) {
		if (child instanceof HTMLElement && child.classList.contains('grid__row-expand')) {
			return child;
		}
	}
	return null;
};

const getDirectParentRow = (nestedRows: HTMLElement): HTMLElement | null => {
	for (const child of Array.from(nestedRows.children)) {
		if (child instanceof HTMLElement && child.classList.contains('grid__row--parent')) {
			return child;
		}
	}
	return null;
};

const getWrappedGridRow = (wrapper: HTMLElement): HTMLElement | null => {
	for (const child of Array.from(wrapper.children)) {
		if (!(child instanceof HTMLElement)) {
			continue;
		}
		if (child.classList.contains('grid__row')) {
			return child;
		}
		if (child.classList.contains('grid__nested-rows')) {
			return getDirectParentRow(child);
		}
	}
	return null;
};

const isGridNavigationKey = (key: string): key is GridNavigationKey => {
	return gridNavigationKeys.includes(key as GridNavigationKey);
};

const moveGridRow = (
	row: HTMLElement,
	direction: -1 | 1,
	onMoveRow: (overRowID: string, movedRowID: string) => void,
) => {
	const wrapper = row.closest<HTMLElement>('.draggable-wrapper');
	if (wrapper?.parentElement == null) {
		return;
	}
	const siblings = Array.from(wrapper.parentElement.children).filter(
		(child): child is HTMLElement => child instanceof HTMLElement && child.classList.contains('draggable-wrapper'),
	);
	const currentIndex = siblings.indexOf(wrapper);
	const target = siblings[currentIndex + direction];
	if (target == null) {
		return;
	}
	const targetRow = getWrappedGridRow(target);
	const movedRowID = row.dataset.id;
	const overRowID = targetRow?.dataset.id;
	if (movedRowID == null || overRowID == null) {
		return;
	}
	onMoveRow(overRowID, movedRowID);
};

const selectGridRow = (row: HTMLElement) => {
	row.scrollIntoView({ block: 'nearest' });
	row.click();
};

export { focusGridForKeyboardNavigation, navigateGrid, navigateGridWithKeyboard, shouldNavigateGridFromDocument };
