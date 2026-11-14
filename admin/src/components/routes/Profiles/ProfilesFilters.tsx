import React, { useCallback, useEffect, useId, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { createPortal, flushSync } from 'react-dom';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaButtonElement from '@awesome.me/webawesome/dist/components/button/button.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import WaTooltip from '@awesome.me/webawesome/dist/react/tooltip/index.js';
import { FilterEditor } from '../../base/FilterEditor/FilterEditor';
import { Filter, FilterRule } from '../../../lib/api/types/pipeline';
import { ObjectType } from '../../../lib/api/types/types';
import { getFilterConditions, isFilterGroup, validateAndNormalizeFilter } from '../../../lib/core/pipeline';
import { serializeFilter } from '../../../utils/filters';
import { sleep } from '../../../utils/sleep';
import { getPropertyPathDisplayValue } from '../../helpers/getPropertyPathDisplayValue';
import { ProfileSchemaProperties } from './Profiles.types';
import { getProfilesFilterErrors } from './profilesFilterValidation';
import {
	cloneProfileFilter,
	commitProfilesFilter,
	createProfileFilterHistoryEntry,
	createProfilesFilterSession,
	currentProfilesFilterHistoryEntry,
	FilterHistoryEntry,
	FilterSession,
	profileFilterFingerprint,
	profileFilterResultsFingerprint,
} from './profilesFilterSession';

const MIN_PROFILES_LOADING_DURATION_MS = 400;

interface ValidatedDraft {
	filter?: Filter | null;
	error?: string;
}

interface CollapsedFilterItem {
	fullLabel: string;
	label: string;
	type: 'condition' | 'group';
}

const countLeafFilterConditions = (rule: FilterRule): number =>
	isFilterGroup(rule)
		? rule.rules.reduce((count, nestedRule) => count + countLeafFilterConditions(nestedRule), 0)
		: 1;

const formatFilterConditionCount = (count: number): string => `${count} ${count === 1 ? 'condition' : 'conditions'}`;

const collapsedFilterRuleLabel = (rule: FilterRule, propertyLabel: (property: string) => string): string => {
	if (!isFilterGroup(rule)) {
		return serializeFilter({ operator: 'and', rules: [rule] }, false, propertyLabel);
	}

	return rule.rules
		.map((nestedRule) => {
			const label = collapsedFilterRuleLabel(nestedRule, propertyLabel);
			return isFilterGroup(nestedRule) ? `(${label})` : label;
		})
		.join(` ${rule.operator} `);
};

const createCollapsedFilterItem = (
	rule: FilterRule,
	propertyLabel: (property: string) => string,
): CollapsedFilterItem => {
	const fullLabel = collapsedFilterRuleLabel(rule, propertyLabel);
	if (!isFilterGroup(rule)) {
		return { fullLabel, label: fullLabel, type: 'condition' };
	}

	const isSimpleGroup = rule.rules.length <= 2 && rule.rules.every((nestedRule) => !isFilterGroup(nestedRule));
	if (isSimpleGroup) {
		return { fullLabel, label: fullLabel, type: 'group' };
	}

	const conditionCount = countLeafFilterConditions(rule);
	const logical = rule.operator === 'and' ? 'all' : 'any';

	return {
		fullLabel,
		label: `${logical} group · ${formatFilterConditionCount(conditionCount)}`,
		type: 'group',
	};
};

interface CollapsedFilterChipLabelProps {
	fullLabel: string;
	label: string;
}

const CollapsedFilterChipLabel = ({ fullLabel, label }: CollapsedFilterChipLabelProps) => {
	const labelID = useId();
	const labelRef = useRef<HTMLSpanElement>(null);
	const [isTruncated, setIsTruncated] = useState(false);

	useLayoutEffect(() => {
		const element = labelRef.current;
		if (element == null) {
			return;
		}

		const updateIsTruncated = () => {
			const truncated = element.scrollWidth > element.clientWidth;
			setIsTruncated((current) => (current === truncated ? current : truncated));
		};

		updateIsTruncated();
		const resizeObserver = new ResizeObserver(updateIsTruncated);
		resizeObserver.observe(element);

		return () => resizeObserver.disconnect();
	}, [label]);

	return (
		<span className='profiles-list__filter-summary-chip-label-container'>
			<WaTooltip className='app-tooltip' for={labelID} disabled={!isTruncated}>
				{fullLabel}
			</WaTooltip>
			<span id={labelID} ref={labelRef} className='profiles-list__filter-summary-chip-label'>
				{label}
			</span>
		</span>
	);
};

interface CollapsedFilterChipsProps {
	disabled: boolean;
	filter: Filter;
	onRemove: (ruleIndex: number) => void;
	propertyLabel: (property: string) => string;
	summary: string;
}

const CollapsedFilterChips = ({ disabled, filter, onRemove, propertyLabel, summary }: CollapsedFilterChipsProps) => {
	const items = useMemo(
		() => filter.rules.map((rule) => createCollapsedFilterItem(rule, propertyLabel)),
		[filter, propertyLabel],
	);
	const conditionCount = filter.rules.reduce((count, rule) => count + countLeafFilterConditions(rule), 0);
	const [visibleItemCount, setVisibleItemCount] = useState(items.length);
	const containerRef = useRef<HTMLDivElement>(null);
	const measurementRef = useRef<HTMLDivElement>(null);
	const overflowMeasurementRef = useRef<HTMLSpanElement>(null);
	const overflowMeasurementLabelRef = useRef<HTMLSpanElement>(null);

	useLayoutEffect(() => {
		const container = containerRef.current;
		const measurement = measurementRef.current;
		const overflowMeasurement = overflowMeasurementRef.current;
		const overflowMeasurementLabel = overflowMeasurementLabelRef.current;
		if (
			container == null ||
			measurement == null ||
			overflowMeasurement == null ||
			overflowMeasurementLabel == null
		) {
			return;
		}

		const updateVisibleItemCount = () => {
			const itemWidths = Array.from(measurement.children, (item) => (item as HTMLElement).offsetWidth);
			const gap = Number.parseFloat(getComputedStyle(measurement).columnGap) || 0;
			const minimumReadableChipWidth = Number.parseFloat(
				getComputedStyle(container).getPropertyValue('--profiles-filter-summary-chip-min-readable-width'),
			);
			const measureOverflowWidth = (omittedCount: number) => {
				overflowMeasurementLabel.textContent = `+${omittedCount} more`;

				return overflowMeasurement.offsetWidth;
			};
			const allItemsWidth =
				itemWidths.reduce((total, width) => total + width, 0) + gap * Math.max(0, itemWidths.length - 1);
			if (allItemsWidth <= container.clientWidth) {
				setVisibleItemCount(items.length);
				return;
			}

			let fittedItemCount = 0;
			let fittedItemsWidth = 0;
			for (let index = 0; index < itemWidths.length - 1; index++) {
				fittedItemsWidth += itemWidths[index];
				const candidateCount = index + 1;
				const requiredWidth =
					fittedItemsWidth + gap * candidateCount + measureOverflowWidth(items.length - candidateCount);
				if (requiredWidth > container.clientWidth) {
					break;
				}
				fittedItemCount = candidateCount;
			}

			if (fittedItemCount === 0 && itemWidths.length > 0) {
				let availableWidth = container.clientWidth;
				if (items.length > 1) {
					availableWidth -= gap + measureOverflowWidth(items.length - 1);
				}
				const requiredWidth = Math.min(itemWidths[0], minimumReadableChipWidth);
				if (availableWidth >= requiredWidth) {
					fittedItemCount = 1;
				}
			}
			setVisibleItemCount(fittedItemCount);
		};

		updateVisibleItemCount();
		const resizeObserver = new ResizeObserver(updateVisibleItemCount);
		resizeObserver.observe(container);

		return () => resizeObserver.disconnect();
	}, [items]);

	const fittedItemCount = Math.min(visibleItemCount, items.length);
	const omittedItemCount = items.length - fittedItemCount;

	const renderItem = (item: CollapsedFilterItem, index: number, removable: boolean) => (
		<span className='profiles-list__filter-summary-unit' key={`${index}:${item.fullLabel}`}>
			{index > 0 && <span className='profiles-list__filter-summary-connector'>{filter.operator}</span>}
			<span className='profiles-list__filter-summary-chip' role='group' aria-label={item.fullLabel}>
				{removable ? (
					<CollapsedFilterChipLabel fullLabel={item.fullLabel} label={item.label} />
				) : (
					<span className='profiles-list__filter-summary-chip-label-container'>
						<span className='profiles-list__filter-summary-chip-label'>{item.label}</span>
					</span>
				)}
				{removable ? (
					<button
						type='button'
						className='profiles-list__filter-summary-chip-remove'
						aria-label={`Remove ${item.type}: ${item.fullLabel}`}
						disabled={disabled}
						onClick={() => onRemove(index)}
					>
						<WaIcon name='x-lg' aria-hidden='true' />
					</button>
				) : (
					<span className='profiles-list__filter-summary-chip-remove-measurement' aria-hidden='true'>
						<WaIcon name='x-lg' />
					</span>
				)}
			</span>
		</span>
	);

	return (
		<div
			ref={containerRef}
			className='profiles-list__filter-summary-chips'
			role='group'
			aria-label={`Applied filters: ${summary}`}
			aria-busy={disabled}
		>
			<div ref={measurementRef} className='profiles-list__filter-summary-measurement' aria-hidden='true'>
				{items.map((item, index) => renderItem(item, index, false))}
			</div>
			<span
				ref={overflowMeasurementRef}
				className='profiles-list__filter-summary-overflow-measurement'
				aria-hidden='true'
			>
				<span className='profiles-list__filter-summary-connector'>{filter.operator}</span>
				<span ref={overflowMeasurementLabelRef} className='profiles-list__filter-summary-overflow'>
					+{items.length} more
				</span>
			</span>
			<div className='profiles-list__filter-summary-visible-chips'>
				{items.slice(0, fittedItemCount).map((item, index) => renderItem(item, index, true))}
				{fittedItemCount > 0 && omittedItemCount > 0 && (
					<span className='profiles-list__filter-summary-overflow-unit'>
						<span className='profiles-list__filter-summary-connector'>{filter.operator}</span>
						<span className='profiles-list__filter-summary-overflow'>+{omittedItemCount} more</span>
					</span>
				)}
				{fittedItemCount === 0 && items.length > 0 && (
					<span className='profiles-list__filter-summary-fallback'>
						{formatFilterConditionCount(conditionCount)}
					</span>
				)}
			</div>
		</div>
	);
};

interface ProfilesFiltersProps {
	appliedFilter: Filter | null;
	gridActionContainer: HTMLDivElement | null;
	onInspectProfiles: () => void;
	onShow: (filter: Filter | null) => Promise<boolean>;
	schema: ObjectType;
	schemaProperties: ProfileSchemaProperties;
}

const ProfilesFilters = ({
	appliedFilter,
	gridActionContainer,
	onInspectProfiles,
	onShow,
	schema,
	schemaProperties,
}: ProfilesFiltersProps) => {
	const [session, setSession] = useState<FilterSession>(() => createProfilesFilterSession(appliedFilter));
	const [isOpen, setIsOpen] = useState(false);
	const [isShortcutActive, setIsShortcutActive] = useState(true);
	const [pendingGridContext, setPendingGridContext] = useState<string>();
	const [hasGridFailure, setHasGridFailure] = useState(false);
	const [resultsUpdatedContext, setResultsUpdatedContext] = useState<string>();
	const [validationAttempt, setValidationAttempt] = useState(0);
	const sessionRef = useRef(session);
	const gridRequestGenerationRef = useRef(0);
	const draftRevisionRef = useRef(0);
	const expectedDisplayedContextRef = useRef<string>();
	const displayedContextRef = useRef<string>();
	const panelRef = useRef<HTMLDivElement>(null);
	const editorRef = useRef<HTMLElement>(null);
	const triggerRef = useRef<WaButtonElement>(null);
	const isMac = navigator.platform.startsWith('Mac');

	const updateSession = useCallback((update: (current: FilterSession) => FilterSession) => {
		const next = update(sessionRef.current);
		sessionRef.current = next;
		setSession(next);
	}, []);

	const updateSessionWithHistoryTransition = useCallback((update: (current: FilterSession) => FilterSession) => {
		const next = update(sessionRef.current);
		sessionRef.current = next;
		if (
			typeof document.startViewTransition !== 'function' ||
			window.matchMedia('(prefers-reduced-motion: reduce)').matches
		) {
			setSession(next);
			return;
		}
		document.startViewTransition(() => flushSync(() => setSession(sessionRef.current)));
	}, []);

	useEffect(() => {
		if (resultsUpdatedContext == null) {
			return;
		}
		const timeoutID = window.setTimeout(() => setResultsUpdatedContext(undefined), 1200);
		return () => window.clearTimeout(timeoutID);
	}, [resultsUpdatedContext]);

	const normalizeFilter = useCallback(
		(filter: Filter | null): Filter | null => {
			if (filter == null) {
				return null;
			}
			const normalized = validateAndNormalizeFilter(filter, schema);
			// A single empty condition represents an unfiltered view.
			if (normalized == null && filter.rules.length === 1 && !isFilterGroup(filter.rules[0])) {
				return null;
			}
			const draftConditions = getFilterConditions(filter);
			const conditions = normalized == null ? [] : getFilterConditions(normalized);
			// Empty editor placeholders belong to the draft, not to a complete version.
			if (
				draftConditions.length !== conditions.length ||
				draftConditions.some(
					(condition, index) => (condition.values?.length ?? 0) !== (conditions[index].values?.length ?? 0),
				)
			) {
				throw new Error('Complete the filter before applying it');
			}
			return normalized;
		},
		[schema],
	);

	const validatedDraft = useMemo<ValidatedDraft>(() => {
		try {
			return { filter: normalizeFilter(session.draft) };
		} catch (error) {
			return { error: error instanceof Error ? error.message : String(error) };
		}
	}, [normalizeFilter, session.draft]);
	const validationErrors = useMemo(
		() =>
			validationAttempt > 0 && validatedDraft.error != null ? getProfilesFilterErrors(session.draft, schema) : {},
		[schema, session.draft, validatedDraft.error, validationAttempt],
	);
	const validateDraft = useCallback((): Filter | null | undefined => {
		try {
			return normalizeFilter(sessionRef.current.draft);
		} catch {
			setValidationAttempt((attempt) => attempt + 1);
			return undefined;
		}
	}, [normalizeFilter]);

	const historyEntry = currentProfilesFilterHistoryEntry(session);
	const hasFilters = validatedDraft.filter != null || validatedDraft.error != null;
	const rawDraftFingerprint = profileFilterFingerprint(session.draft);
	const isDraftDirty = rawDraftFingerprint !== profileFilterFingerprint(historyEntry.draft);
	const canUndo = isDraftDirty || session.historyIndex > 0;
	const canRedo = session.historyIndex < session.history.length - 1;
	const appliedFilterFingerprint = profileFilterFingerprint(appliedFilter);
	const hasUnappliedChanges =
		validatedDraft.error != null ||
		profileFilterResultsFingerprint(validatedDraft.filter ?? null) !==
			profileFilterResultsFingerprint(appliedFilter);
	const isUpdatingGridResults = pendingGridContext != null;
	const canApplyFilters = hasUnappliedChanges && !isUpdatingGridResults;
	const canApplyWithShortcut = canApplyFilters && isShortcutActive;
	const propertyLabel = useCallback(
		(property: string) => getPropertyPathDisplayValue(property, (path) => schemaProperties[path]),
		[schemaProperties],
	);
	const summary = useMemo(
		() => (appliedFilter == null ? 'No filters applied' : serializeFilter(appliedFilter, false, propertyLabel)),
		[appliedFilter, propertyLabel],
	);

	const onDraftChange = useCallback(
		(draft: Filter | null) => {
			if (profileFilterFingerprint(sessionRef.current.draft) === profileFilterFingerprint(draft)) {
				return;
			}
			draftRevisionRef.current++;
			updateSession((state) => ({ ...state, draft }));
			setHasGridFailure(false);
		},
		[updateSession],
	);

	const onDraftCommit = useCallback(
		(draft: Filter | null) => {
			let filter: Filter | null;
			try {
				filter = normalizeFilter(draft);
			} catch {
				return;
			}
			updateSession((state) => commitProfilesFilter(state, filter, draft));
		},
		[normalizeFilter, updateSession],
	);

	const selectHistoryEntry = (historyIndex: number) => {
		const target = sessionRef.current.history[historyIndex];
		draftRevisionRef.current++;
		setHasGridFailure(false);
		updateSessionWithHistoryTransition((state) => ({
			...state,
			draft: cloneProfileFilter(target.draft),
			historyIndex,
		}));
	};

	const undo = () => {
		const current = sessionRef.current;
		const entry = currentProfilesFilterHistoryEntry(current);
		if (profileFilterFingerprint(current.draft) !== profileFilterFingerprint(entry.draft)) {
			selectHistoryEntry(current.historyIndex);
			return;
		}
		if (current.historyIndex === 0) {
			return;
		}
		selectHistoryEntry(current.historyIndex - 1);
	};

	const redo = () => {
		const current = sessionRef.current;
		if (current.historyIndex >= current.history.length - 1) {
			return;
		}
		selectHistoryEntry(current.historyIndex + 1);
	};

	const restoreTriggerFocus = useCallback(() => {
		requestAnimationFrame(() => triggerRef.current?.focus({ preventScroll: true }));
	}, []);

	const setEditorOpen = useCallback(
		(open: boolean) => {
			setIsOpen(open);
			if (open) {
				setValidationAttempt(0);
				setIsShortcutActive(true);
				triggerRef.current?.blur();
				return;
			}
			restoreTriggerFocus();
		},
		[restoreTriggerFocus],
	);

	const materializeProfiles = useCallback(
		async (entry: FilterHistoryEntry, shouldInspect = true) => {
			const snapshot = createProfileFilterHistoryEntry(entry.filter, entry.draft);
			const snapshotContext = snapshot.fingerprint;
			const generation = ++gridRequestGenerationRef.current;
			const draftRevision = draftRevisionRef.current;
			expectedDisplayedContextRef.current = snapshotContext;
			setPendingGridContext(snapshotContext);
			setHasGridFailure(false);
			setResultsUpdatedContext(undefined);
			const minimumLoading = sleep(MIN_PROFILES_LOADING_DURATION_MS);
			try {
				const shown = await onShow(snapshot.filter);
				await minimumLoading;
				if (gridRequestGenerationRef.current !== generation) {
					return;
				}
				setPendingGridContext(undefined);
				if (!shown) {
					if (expectedDisplayedContextRef.current === snapshotContext) {
						expectedDisplayedContextRef.current = undefined;
					}
					return;
				}
				setResultsUpdatedContext(snapshotContext);
				let currentDraftFingerprint: string | undefined;
				try {
					currentDraftFingerprint = profileFilterFingerprint(normalizeFilter(sessionRef.current.draft));
				} catch {
					currentDraftFingerprint = undefined;
				}
				const intentIsCurrent =
					draftRevisionRef.current === draftRevision && currentDraftFingerprint === snapshot.fingerprint;
				if (shouldInspect && intentIsCurrent) {
					onInspectProfiles();
				}
			} catch {
				await minimumLoading;
				if (gridRequestGenerationRef.current === generation) {
					expectedDisplayedContextRef.current = undefined;
					setPendingGridContext(undefined);
					setHasGridFailure(true);
				}
			}
		},
		[normalizeFilter, onInspectProfiles, onShow],
	);

	const removeAppliedFilterRule = useCallback(
		(ruleIndex: number) => {
			if (appliedFilter == null || pendingGridContext != null || ruleIndex >= appliedFilter.rules.length) {
				return;
			}

			const nextFilter = cloneProfileFilter(appliedFilter);
			if (nextFilter == null) {
				return;
			}
			nextFilter.rules.splice(ruleIndex, 1);
			const filter = nextFilter.rules.length === 0 ? null : nextFilter;
			const entry = createProfileFilterHistoryEntry(filter);
			draftRevisionRef.current++;
			updateSession((state) => commitProfilesFilter(state, filter, entry.draft));
			void materializeProfiles(entry, false);
		},
		[appliedFilter, materializeProfiles, pendingGridContext, updateSession],
	);

	const collapseFilters = useCallback(() => {
		const draft = sessionRef.current.draft;
		const filter = validateDraft();
		if (filter === undefined) {
			return;
		}
		draftRevisionRef.current++;
		updateSession((state) => commitProfilesFilter(state, filter, draft));
		setEditorOpen(false);
		const entry = currentProfilesFilterHistoryEntry(sessionRef.current);
		if (pendingGridContext === entry.fingerprint) {
			return;
		}
		if (
			pendingGridContext == null &&
			profileFilterResultsFingerprint(filter) === profileFilterResultsFingerprint(appliedFilter)
		) {
			return;
		}
		void materializeProfiles(entry, false);
	}, [appliedFilter, materializeProfiles, pendingGridContext, setEditorOpen, updateSession, validateDraft]);

	const displayedContext = appliedFilterFingerprint;
	useEffect(() => {
		if (displayedContextRef.current === displayedContext) {
			return;
		}
		displayedContextRef.current = displayedContext;
		if (expectedDisplayedContextRef.current === displayedContext) {
			expectedDisplayedContextRef.current = undefined;
			return;
		}
		draftRevisionRef.current++;
		updateSession(() => createProfilesFilterSession(appliedFilter));
	}, [appliedFilter, displayedContext, updateSession]);

	useEffect(
		() => () => {
			gridRequestGenerationRef.current++;
		},
		[],
	);

	useEffect(() => {
		if (!isOpen) {
			return;
		}
		const onKeyDown = (event: globalThis.KeyboardEvent) => {
			if (
				event.key !== 'Escape' ||
				event.defaultPrevented ||
				!event.composedPath().includes(panelRef.current as EventTarget)
			) {
				return;
			}
			const popupIsOpen = event.composedPath().some((element) => {
				if (!(element instanceof HTMLElement)) {
					return false;
				}
				return (
					element.classList.contains('combobox--open') ||
					(element.tagName === 'WA-SELECT' && (element as any).open)
				);
			});
			if (popupIsOpen) {
				return;
			}
			event.preventDefault();
			if (!hasUnappliedChanges) {
				setEditorOpen(false);
			}
		};
		document.addEventListener('keydown', onKeyDown, true);
		return () => document.removeEventListener('keydown', onKeyDown, true);
	}, [hasUnappliedChanges, isOpen, setEditorOpen]);

	const applyFilters = useCallback(() => {
		if (!canApplyFilters) {
			return;
		}
		const draft = sessionRef.current.draft;
		const filter = validateDraft();
		if (filter === undefined) {
			return;
		}
		updateSession((state) => commitProfilesFilter(state, filter, draft));
		void materializeProfiles(currentProfilesFilterHistoryEntry(sessionRef.current));
	}, [canApplyFilters, materializeProfiles, updateSession, validateDraft]);

	useEffect(() => {
		const editor = editorRef.current;
		if (!isOpen || editor == null) {
			return;
		}
		// The grid itself can receive a passive click without selecting a profile.
		const isActionTarget = (target: EventTarget) =>
			target instanceof HTMLElement &&
			(target.matches('.grid__row--clickable, .drawer-attributes--parent') ||
				(target.tabIndex >= 0 && !target.classList.contains('profiles-list__grid')));
		const onClick = (event: MouseEvent) => {
			const path = event.composedPath();
			// This listener can be added while the opening click is still propagating.
			if (path.includes(triggerRef.current as EventTarget)) {
				return;
			}
			setIsShortcutActive(path.includes(editor) || (!event.defaultPrevented && !path.some(isActionTarget)));
		};
		const onFocusIn = (event: FocusEvent) => {
			setIsShortcutActive(event.composedPath().includes(editor));
		};
		const onKeyDown = (event: KeyboardEvent) => {
			const hasApplyModifier = isMac ? event.metaKey : event.ctrlKey;
			const isApplyShortcut =
				event.key === 'Enter' && hasApplyModifier && !event.altKey && !event.shiftKey && !event.isComposing;
			if (!isApplyShortcut) {
				if (
					!event.composedPath().includes(editor) &&
					!['Alt', 'Control', 'Meta', 'Shift'].includes(event.key)
				) {
					setIsShortcutActive(false);
				}
				return;
			}
			if (!isShortcutActive) {
				return;
			}
			event.preventDefault();
			event.stopPropagation();
			if (!event.repeat) {
				applyFilters();
			}
		};
		document.addEventListener('click', onClick);
		document.addEventListener('focusin', onFocusIn);
		document.addEventListener('keydown', onKeyDown, true);
		return () => {
			document.removeEventListener('click', onClick);
			document.removeEventListener('focusin', onFocusIn);
			document.removeEventListener('keydown', onKeyDown, true);
		};
	}, [applyFilters, isMac, isOpen, isShortcutActive]);

	const gridErrorMessage = hasGridFailure && (
		<span className='profiles-list__filter-grid-error' role='alert'>
			Unable to apply filters
		</span>
	);

	// History actions discard the focused draft instead of confirming it on blur.
	const historyActions = (
		<>
			<WaButton
				className='profiles-list__filter-history-action'
				size='s'
				variant='brand'
				appearance='plain'
				disabled={!canUndo}
				onPointerDown={(event) => event.preventDefault()}
				onClick={undo}
			>
				<WaIcon slot='start' name='arrow-counterclockwise' aria-hidden='true' />
				Undo
			</WaButton>
			<WaButton
				className='profiles-list__filter-history-action'
				size='s'
				variant='brand'
				appearance='plain'
				disabled={!canRedo}
				onPointerDown={(event) => event.preventDefault()}
				onClick={redo}
			>
				<WaIcon slot='start' name='arrow-clockwise' aria-hidden='true' />
				Redo
			</WaButton>
		</>
	);

	return (
		<div
			ref={panelRef}
			className={`profiles-list__filters${isOpen ? ' profiles-list__filters--expanded' : ''}${isUpdatingGridResults ? ' profiles-list__filters--updating-results' : ''}${resultsUpdatedContext != null ? ' profiles-list__filters--results-updated' : ''}`}
		>
			<div className='profiles-list__filter-summary'>
				<div className='profiles-list__filter-heading'>
					<span className='profiles-list__filter-title'>Filters</span>
					<WaButton
						ref={triggerRef}
						className='profiles-list__filter-toggle'
						size='s'
						appearance='outlined'
						aria-controls='profiles-filter-editor'
						aria-expanded={isOpen}
						onClick={isOpen ? collapseFilters : () => setEditorOpen(true)}
					>
						<WaIcon
							slot='start'
							name={isOpen ? 'chevron-up' : hasFilters ? 'sliders2' : 'plus'}
							aria-hidden='true'
						/>
						{isOpen ? 'Collapse' : hasFilters ? 'Edit filters' : 'Add filter'}
					</WaButton>
				</div>
				<div
					ref={(element) => {
						if (element != null) {
							element.inert = isOpen;
						}
					}}
					className='profiles-list__filter-summary-content'
					aria-hidden={isOpen}
				>
					{appliedFilter == null ? (
						<span className='profiles-list__filter-summary-value'>{summary}</span>
					) : (
						<CollapsedFilterChips
							disabled={pendingGridContext != null}
							filter={appliedFilter}
							onRemove={removeAppliedFilterRule}
							propertyLabel={propertyLabel}
							summary={summary}
						/>
					)}
				</div>
			</div>
			<div
				ref={(element) => {
					if (element != null) {
						element.inert = !isOpen;
					}
				}}
				className='profiles-list__filter-editor-disclosure'
				aria-hidden={!isOpen}
			>
				<div className='profiles-list__filter-disclosure-clip'>
					<div className='profiles-list__filter-expanded-content'>
						<section
							ref={editorRef}
							id='profiles-filter-editor'
							className='profiles-list__filter-editor'
							role='region'
							aria-label='Profile filters'
							tabIndex={-1}
						>
							<div className='profiles-list__filter-editor-body'>
								<FilterEditor
									errors={validationErrors}
									filter={session.draft}
									hoistMenus={true}
									onChange={onDraftChange}
									onCommit={onDraftCommit}
									role='Destination'
									rootActions={historyActions}
									schema={schema}
									showEmptyFilterAsCondition={true}
									subject='Profiles'
									target='User'
									validationAttempt={validationAttempt}
								/>
							</div>
							<div className='profiles-list__filter-options'>
								{isOpen && gridErrorMessage}
								<WaButton
									className='profiles-list__filter-apply'
									variant={canApplyFilters || isUpdatingGridResults ? 'brand' : 'neutral'}
									appearance='accent'
									size='m'
									aria-keyshortcuts={isMac ? 'Meta+Enter' : 'Control+Enter'}
									disabled={!canApplyFilters}
									loading={isUpdatingGridResults}
									onClick={applyFilters}
								>
									Apply filters
								</WaButton>
								<span
									className={`profiles-list__filter-shortcut grid-keyboard-hints__keys${canApplyWithShortcut ? '' : ' grid-keyboard-hints__hint--disabled'}`}
									aria-disabled={!canApplyWithShortcut || undefined}
								>
									<span className='profiles-list__filter-shortcut-label'>Shortcut:</span>
									<kbd>{isMac ? '⌘' : 'Ctrl'}</kbd>
									<span className='grid-keyboard-hints__plus'>+</span>
									<kbd>Enter</kbd>
								</span>
							</div>
						</section>
						<div className='profiles-list__filter-preview-context'>
							<div className='profiles-list__filter-overview'>
								<div className='profiles-list__filter-description'>
									Choose which profiles to include in the results. Leave empty to include all
									profiles.
								</div>
								<a
									className='profiles-list__filter-documentation'
									href='https://www.krenalis.com/docs/ref/admin/filters'
									target='_blank'
									rel='noopener'
								>
									Learn more about filters
								</a>
							</div>
						</div>
					</div>
				</div>
			</div>
			{gridActionContainer != null &&
				createPortal(
					<>
						{!isOpen && gridErrorMessage}
						{isUpdatingGridResults ? (
							<span className='profiles-list__filter-status' role='status'>
								Updating profiles…
							</span>
						) : !isOpen && !hasGridFailure && hasUnappliedChanges ? (
							<span className='profiles-list__filter-status' role='status'>
								Unapplied changes · Showing previous results
							</span>
						) : null}
					</>,
					gridActionContainer,
				)}
		</div>
	);
};

export { ProfilesFilters };
