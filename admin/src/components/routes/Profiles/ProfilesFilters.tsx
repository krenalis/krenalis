import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { createPortal, flushSync } from 'react-dom';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlButtonElement from '@shoelace-style/shoelace/dist/components/button/button.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { FilterEditor } from '../../base/FilterEditor/FilterEditor';
import { Filter, FilterRule } from '../../../lib/api/types/pipeline';
import { CountProfilesResponse } from '../../../lib/api/types/responses';
import { ObjectType } from '../../../lib/api/types/types';
import { isFilterGroup, validateAndNormalizeFilter } from '../../../lib/core/pipeline';
import { formatNumber } from '../../../utils/formatNumber';
import { serializeFilter } from '../../../utils/filters';
import { getProfilePropertyPathLabel } from './Profiles.helpers';
import { ProfileSchemaProperties } from './Profiles.types';
import {
	cloneProfileFilter,
	commitProfilesFilterPreview,
	createProfileFilterHistoryEntry,
	createProfilesFilterSession,
	currentProfilesFilterHistoryEntry,
	FilterHistoryEntry,
	FilterSession,
	PreviewCandidate,
	profileFilterFingerprint,
	profileFilterResultsFingerprint,
} from './profilesFilterSession';
import { createProfilesPreviewTiming, recordProfilesPreviewLatency } from './profilesPreviewTiming';

interface ProfilesFiltersProps {
	appliedFilter: Filter | null;
	gridActionContainer: HTMLDivElement | null;
	onInspectProfiles: () => void;
	onPreview: (filter: Filter | null, signal?: AbortSignal) => Promise<CountProfilesResponse>;
	onShow: (filter: Filter | null) => Promise<boolean>;
	profilesTotal?: number;
	schema: ObjectType;
	schemaProperties: ProfileSchemaProperties;
}

interface ValidatedDraft {
	filter?: Filter | null;
	error?: string;
}

interface FailedGridRequest {
	entry: FilterHistoryEntry;
	intent: GridMaterializationIntent;
	message: string;
}

type GridMaterializationIntent = 'collapse' | 'remove' | 'show';

interface ExpectedDisplayedContext {
	context: string;
	intent: GridMaterializationIntent;
	session: FilterSession;
}

interface CollapsedFilterChipsProps {
	disabled: boolean;
	filter: Filter;
	onRemove: (ruleIndex: number) => void;
	propertyLabel: (property: string) => string;
	summary: string;
}

interface CollapsedFilterItem {
	fullLabel: string;
	label: string;
	type: 'condition' | 'group';
}

interface CollapsedFilterChipLabelProps {
	fullLabel: string;
	label: string;
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

const CollapsedFilterChipLabel = ({ fullLabel, label }: CollapsedFilterChipLabelProps) => {
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
		<SlTooltip
			className='app-tooltip profiles-list__filter-summary-chip-label-container'
			content={fullLabel}
			disabled={!isTruncated}
			hoist={true}
		>
			<span ref={labelRef} className='profiles-list__filter-summary-chip-label'>
				{label}
			</span>
		</SlTooltip>
	);
};

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
						<SlIcon name='x-lg' aria-hidden='true' />
					</button>
				) : (
					<span className='profiles-list__filter-summary-chip-remove-measurement' aria-hidden='true'>
						<SlIcon name='x-lg' />
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

const ProfilesFilters = ({
	appliedFilter,
	gridActionContainer,
	onInspectProfiles,
	onPreview,
	onShow,
	profilesTotal,
	schema,
	schemaProperties,
}: ProfilesFiltersProps) => {
	const [session, setSession] = useState<FilterSession>(() =>
		createProfilesFilterSession(appliedFilter, profilesTotal),
	);
	const [isOpen, setIsOpen] = useState(false);
	const [pendingGridContext, setPendingGridContext] = useState<string>();
	const [gridFailure, setGridFailure] = useState<FailedGridRequest>();
	const [resultsUpdatedContext, setResultsUpdatedContext] = useState<string>();
	const [previewTiming, setPreviewTiming] = useState(createProfilesPreviewTiming);
	const sessionRef = useRef(session);
	const previewGenerationRef = useRef(0);
	const previewControllerRef = useRef<AbortController>();
	const visibilityGenerationRef = useRef(0);
	const gridRequestGenerationRef = useRef(0);
	const draftRevisionRef = useRef(0);
	const expectedDisplayedContextRef = useRef<ExpectedDisplayedContext>();
	const displayedContextRef = useRef<string>();
	const panelRef = useRef<HTMLDivElement>(null);
	const triggerRef = useRef<SlButtonElement>(null);

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
		document.startViewTransition(() => flushSync(() => setSession(next)));
	}, []);

	useEffect(() => {
		if (resultsUpdatedContext == null) {
			return;
		}
		const timeoutID = window.setTimeout(() => setResultsUpdatedContext(undefined), 1200);
		return () => window.clearTimeout(timeoutID);
	}, [resultsUpdatedContext]);

	const normalizeFilter = useCallback(
		(filter: Filter | null): Filter | null =>
			filter == null ? null : validateAndNormalizeFilter(filter, schema, 'Destination', 'User'),
		[schema],
	);

	const validatedDraft = useMemo<ValidatedDraft>(() => {
		try {
			return { filter: normalizeFilter(session.draft) };
		} catch (error) {
			return { error: error instanceof Error ? error.message : String(error) };
		}
	}, [normalizeFilter, session.draft]);

	const historyEntry = currentProfilesFilterHistoryEntry(session);
	const rawDraftFingerprint = profileFilterFingerprint(session.draft);
	const isDraftDirty = rawDraftFingerprint !== profileFilterFingerprint(historyEntry.filter);
	const normalizedDraftFingerprint =
		validatedDraft.error == null ? profileFilterFingerprint(validatedDraft.filter ?? null) : undefined;
	const hasCurrentPreview = normalizedDraftFingerprint === historyEntry.fingerprint && historyEntry.total != null;
	const draftMatchesCurrentPreview = !isDraftDirty && hasCurrentPreview;
	const isHistoryNavigationPending = session.pending?.historyIndex != null;
	const canUndo = isDraftDirty || (!isHistoryNavigationPending && session.historyIndex > 0);
	const canRedo = !isDraftDirty && session.pending == null && session.historyIndex < session.history.length - 1;
	const appliedFilterFingerprint = profileFilterFingerprint(appliedFilter);
	const previewFilterDiffersFromGrid =
		profileFilterResultsFingerprint(historyEntry.filter) !== profileFilterResultsFingerprint(appliedFilter);
	const propertyLabel = useCallback(
		(property: string) => getProfilePropertyPathLabel(property, schemaProperties),
		[schemaProperties],
	);
	const summary = useMemo(
		() => (appliedFilter == null ? 'No filters' : serializeFilter(appliedFilter, false, propertyLabel)),
		[appliedFilter, propertyLabel],
	);

	const invalidatePreview = useCallback(() => {
		previewGenerationRef.current++;
		previewControllerRef.current?.abort();
		previewControllerRef.current = undefined;
	}, []);

	const previewCandidate = useCallback(
		async (candidate: PreviewCandidate) => {
			const current = sessionRef.current;
			if (current.pending?.fingerprint === candidate.fingerprint) {
				return;
			}

			invalidatePreview();
			const generation = previewGenerationRef.current;
			const controller = new AbortController();
			previewControllerRef.current = controller;
			updateSession((state) => ({ ...state, pending: candidate, failure: undefined }));

			try {
				const visibilityGeneration = visibilityGenerationRef.current;
				const startedVisible = document.visibilityState === 'visible';
				const startedAt = performance.now();
				const response = await onPreview(candidate.filter, controller.signal);
				if (
					!controller.signal.aborted &&
					startedVisible &&
					document.visibilityState === 'visible' &&
					visibilityGenerationRef.current === visibilityGeneration
				) {
					setPreviewTiming((timing) => recordProfilesPreviewLatency(timing, performance.now() - startedAt));
				}
				if (previewGenerationRef.current !== generation) {
					return;
				}
				const update = (state: FilterSession): FilterSession =>
					commitProfilesFilterPreview(state, candidate, response);
				if (candidate.historyIndex == null) {
					updateSession(update);
				} else {
					updateSessionWithHistoryTransition(update);
				}
			} catch (error) {
				if (previewGenerationRef.current !== generation || (error as any)?.name === 'AbortError') {
					return;
				}
				updateSession((state) => ({
					...state,
					pending: undefined,
					failure: { candidate, message: 'Unable to calculate matches.' },
				}));
			} finally {
				if (previewGenerationRef.current === generation) {
					previewControllerRef.current = undefined;
				}
			}
		},
		[invalidatePreview, onPreview, updateSession, updateSessionWithHistoryTransition],
	);

	const onDraftChange = useCallback(
		(draft: Filter | null) => {
			if (profileFilterFingerprint(sessionRef.current.draft) === profileFilterFingerprint(draft)) {
				return;
			}
			draftRevisionRef.current++;
			invalidatePreview();
			updateSession((state) => ({
				...state,
				draft,
				pending: undefined,
				failure: undefined,
			}));
			setGridFailure(undefined);
		},
		[invalidatePreview, updateSession],
	);

	const onDraftCommit = useCallback(
		(draft: Filter | null) => {
			let filter: Filter | null;
			try {
				filter = normalizeFilter(draft);
			} catch {
				return;
			}
			void previewCandidate({ filter, fingerprint: profileFilterFingerprint(filter) });
		},
		[normalizeFilter, previewCandidate],
	);

	const undo = () => {
		const current = sessionRef.current;
		const entry = currentProfilesFilterHistoryEntry(current);
		if (profileFilterFingerprint(current.draft) !== profileFilterFingerprint(entry.filter)) {
			draftRevisionRef.current++;
			invalidatePreview();
			updateSessionWithHistoryTransition((state) => ({
				...state,
				draft: cloneProfileFilter(currentProfilesFilterHistoryEntry(state).filter),
				pending: undefined,
				failure: undefined,
			}));
			return;
		}
		if (current.pending != null || current.historyIndex === 0) {
			return;
		}
		const historyIndex = current.historyIndex - 1;
		const target = current.history[historyIndex];
		setGridFailure(undefined);
		void previewCandidate({ filter: target.filter, fingerprint: target.fingerprint, historyIndex });
	};

	const redo = () => {
		const current = sessionRef.current;
		if (
			current.pending != null ||
			profileFilterFingerprint(current.draft) !==
				profileFilterFingerprint(currentProfilesFilterHistoryEntry(current).filter) ||
			current.historyIndex >= current.history.length - 1
		) {
			return;
		}
		const historyIndex = current.historyIndex + 1;
		const target = current.history[historyIndex];
		setGridFailure(undefined);
		void previewCandidate({ filter: target.filter, fingerprint: target.fingerprint, historyIndex });
	};

	const retryPreview = () => {
		if (sessionRef.current.failure != null) {
			void previewCandidate(sessionRef.current.failure.candidate);
		}
	};

	const restoreTriggerFocus = useCallback(() => {
		requestAnimationFrame(() => triggerRef.current?.focus({ preventScroll: true }));
	}, []);

	const setEditorOpen = useCallback(
		(open: boolean) => {
			setIsOpen(open);
			if (open) {
				triggerRef.current?.blur();
				return;
			}
			restoreTriggerFocus();
		},
		[restoreTriggerFocus],
	);

	const materializeProfiles = useCallback(
		async (entry: FilterHistoryEntry, intent: GridMaterializationIntent) => {
			const snapshot = createProfileFilterHistoryEntry(entry.filter, entry.total);
			const snapshotContext = snapshot.fingerprint;
			const generation = ++gridRequestGenerationRef.current;
			const draftRevision = draftRevisionRef.current;
			expectedDisplayedContextRef.current = { context: snapshotContext, intent, session: sessionRef.current };
			setPendingGridContext(snapshotContext);
			setGridFailure(undefined);
			setResultsUpdatedContext(undefined);
			try {
				const shown = await onShow(snapshot.filter);
				if (gridRequestGenerationRef.current !== generation) {
					return;
				}
				setPendingGridContext(undefined);
				if (!shown) {
					if (expectedDisplayedContextRef.current?.context === snapshotContext) {
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
				if (intent === 'show' && intentIsCurrent) {
					onInspectProfiles();
				} else if (intent === 'collapse' && intentIsCurrent) {
					setEditorOpen(false);
				}
			} catch {
				if (gridRequestGenerationRef.current === generation) {
					expectedDisplayedContextRef.current = undefined;
					setPendingGridContext(undefined);
					setGridFailure({ entry: snapshot, intent, message: 'Unable to show profiles.' });
				}
			}
		},
		[normalizeFilter, onInspectProfiles, onShow, setEditorOpen],
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
			const entry = createProfileFilterHistoryEntry(filter, undefined);
			void materializeProfiles(entry, 'remove');
		},
		[appliedFilter, materializeProfiles, pendingGridContext],
	);

	const collapseFilters = useCallback(() => {
		const currentEntry = currentProfilesFilterHistoryEntry(sessionRef.current);
		if (pendingGridContext === currentEntry.fingerprint) {
			return;
		}
		draftRevisionRef.current++;
		invalidatePreview();
		updateSession((state) => ({
			...state,
			draft: cloneProfileFilter(currentProfilesFilterHistoryEntry(state).filter),
			pending: undefined,
			failure: undefined,
		}));
		setGridFailure(undefined);

		const requiresGridMaterialization = previewFilterDiffersFromGrid;
		if (!requiresGridMaterialization) {
			setEditorOpen(false);
			return;
		}
		void materializeProfiles(currentEntry, 'collapse');
	}, [
		invalidatePreview,
		materializeProfiles,
		pendingGridContext,
		previewFilterDiffersFromGrid,
		setEditorOpen,
		updateSession,
	]);

	const displayedContext = appliedFilterFingerprint;
	useEffect(() => {
		if (displayedContextRef.current === displayedContext) {
			if (
				profilesTotal != null &&
				sessionRef.current.history.length === 1 &&
				currentProfilesFilterHistoryEntry(sessionRef.current).total == null &&
				sessionRef.current.pending == null
			) {
				updateSession((state) => ({
					...state,
					history: [createProfileFilterHistoryEntry(appliedFilter, profilesTotal)],
				}));
			}
			return;
		}
		displayedContextRef.current = displayedContext;
		const expectedDisplayedContext = expectedDisplayedContextRef.current;
		if (expectedDisplayedContext?.context === displayedContext) {
			expectedDisplayedContextRef.current = undefined;
			if (
				expectedDisplayedContext.intent === 'remove' &&
				sessionRef.current === expectedDisplayedContext.session
			) {
				updateSession((state) =>
					commitProfilesFilterPreview(
						{ ...state, draft: cloneProfileFilter(appliedFilter) },
						{ filter: appliedFilter, fingerprint: appliedFilterFingerprint },
						{ total: profilesTotal },
					),
				);
			}
			return;
		}
		invalidatePreview();
		draftRevisionRef.current++;
		setPreviewTiming(createProfilesPreviewTiming());
		updateSession(() => createProfilesFilterSession(appliedFilter, profilesTotal));
	}, [appliedFilter, appliedFilterFingerprint, displayedContext, invalidatePreview, profilesTotal, updateSession]);

	useEffect(
		() => () => {
			invalidatePreview();
			gridRequestGenerationRef.current++;
		},
		[invalidatePreview],
	);

	useEffect(() => {
		const onVisibilityChange = () => {
			visibilityGenerationRef.current++;
		};
		document.addEventListener('visibilitychange', onVisibilityChange);
		return () => document.removeEventListener('visibilitychange', onVisibilityChange);
	}, []);

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
					(element.tagName === 'SL-SELECT' && (element as any).open)
				);
			});
			if (popupIsOpen) {
				return;
			}
			event.preventDefault();
			collapseFilters();
		};
		document.addEventListener('keydown', onKeyDown, true);
		return () => document.removeEventListener('keydown', onKeyDown, true);
	}, [collapseFilters, isOpen]);

	const currentPreviewContext = historyEntry.fingerprint;
	const isShowingCurrentPreview = pendingGridContext === currentPreviewContext;
	const canShowCurrentPreview = draftMatchesCurrentPreview && previewFilterDiffersFromGrid;
	const isCurrentResultsUpdateConfirmed =
		resultsUpdatedContext === currentPreviewContext && session.pending == null && draftMatchesCurrentPreview;
	const isUpdatingGridResults = pendingGridContext != null;
	const textValueCommitDelayMs = previewTiming.suspended ? undefined : previewTiming.delayMs;

	const historyActions = (
		<>
			<SlButton
				className='profiles-list__filter-history-action'
				size='small'
				variant='text'
				disabled={!canUndo}
				onClick={undo}
			>
				<SlIcon slot='prefix' name='arrow-counterclockwise' aria-hidden='true' />
				Undo
			</SlButton>
			<SlButton
				className='profiles-list__filter-history-action'
				size='small'
				variant='text'
				disabled={!canRedo}
				onClick={redo}
			>
				<SlIcon slot='prefix' name='arrow-clockwise' aria-hidden='true' />
				Redo
			</SlButton>
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
					<SlButton
						ref={triggerRef}
						className='profiles-list__filter-toggle'
						size='small'
						variant='default'
						aria-controls='profiles-filter-editor'
						aria-expanded={isOpen}
						disabled={isOpen && isShowingCurrentPreview}
						onClick={isOpen ? collapseFilters : () => setEditorOpen(true)}
					>
						{isOpen ? 'Done' : 'Edit'}
					</SlButton>
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
							id='profiles-filter-editor'
							className='profiles-list__filter-editor'
							role='region'
							aria-label='Profile filters'
						>
							<div className='pipeline__filters profiles-list__filter-editor-body'>
								<FilterEditor
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
									textValueCommitDelayMs={textValueCommitDelayMs}
								/>
								{gridFailure != null && (
									<div className='profiles-list__filter-grid-error' role='alert'>
										<span>{gridFailure.message}</span>
										<SlButton
											size='small'
											variant='text'
											onClick={() =>
												void materializeProfiles(gridFailure.entry, gridFailure.intent)
											}
										>
											Retry
										</SlButton>
									</div>
								)}
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
						{previewTiming.suspended && (
							<span className='profiles-list__filter-preview-slow' role='status' aria-live='polite'>
								Counts are taking longer than usual. Press Enter to update.
							</span>
						)}
						{session.failure != null ? (
							<div className='profiles-list__filter-preview-error' role='status'>
								<span>{session.failure.message}</span>
								<SlButton size='small' variant='text' onClick={retryPreview}>
									Retry
								</SlButton>
							</div>
						) : validatedDraft.error != null ? (
							<span className='profiles-list__filter-preview-incomplete' role='status'>
								Complete the filter to calculate matches
							</span>
						) : null}
						{session.pending != null && (
							<div className='profiles-list__filter-preview-count' role='status' aria-live='polite'>
								<SlSpinner />
								<span>Calculating profiles…</span>
							</div>
						)}
						{!isCurrentResultsUpdateConfirmed && session.pending == null && canShowCurrentPreview ? (
							<SlButton
								className='profiles-list__filter-update-results'
								size='small'
								variant='primary'
								outline
								disabled={isShowingCurrentPreview}
								onClick={() => void materializeProfiles(historyEntry, 'show')}
							>
								{isShowingCurrentPreview && <SlSpinner />}
								Show {formatNumber(historyEntry.total ?? 0)}{' '}
								{historyEntry.total === 1 ? 'match' : 'matches'}
							</SlButton>
						) : null}
					</>,
					gridActionContainer,
				)}
		</div>
	);
};

export { ProfilesFilters };
