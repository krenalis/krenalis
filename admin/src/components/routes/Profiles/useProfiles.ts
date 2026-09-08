import { useCallback, useContext, useEffect, useReducer, useRef, useState } from 'react';
import { flushSync } from 'react-dom';
import AppContext from '../../../context/AppContext';
import { UI_BASE_PATH } from '../../../constants/paths';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { ProfileProperty, ProfileSchemaProperties } from './Profiles.types';
import { ObjectType } from '../../../lib/api/types/types';
import { Filter } from '../../../lib/api/types/pipeline';
import { CountProfilesResponse, FindProfilesResponse } from '../../../lib/api/types/responses';
import { flattenSchema } from '../../../lib/core/pipeline';
import { PROFILES_PROPERTIES_KEY } from '../../../constants/storage';
import { getProfilePropertyPathLabel } from './Profiles.helpers';
import {
	createProfilesQueryKey,
	createProfilesSequenceState,
	createProfilesValueFingerprint,
	getProfilesPage,
	getVisibleProfiles,
	profilesSequenceReducer,
	ProfilesQueryIdentity,
	ProfilesSequenceAction,
} from './profilesSequence';

const DEFAULT_PROFILE_LIMIT = 50;
const COLUMN_VISIBILITY_TRANSITION_CLASS = 'profiles-columns-transition';

let activeColumnVisibilityTransition: ViewTransition | undefined;

interface ProfilesRangeRequest {
	schema: ObjectType;
	filter: Filter | null;
	projection: string[];
}

interface InFlightProfilesRequest {
	background: boolean;
	promise: Promise<boolean>;
}

interface ProfilesReplacement {
	activeProfileID: string;
	animateColumnVisibility?: boolean;
	filter: Filter | null;
	first: number;
	profileProperties: ProfileProperty[];
	projection?: string[];
}

type ProfileNavigationDirection = 'previous' | 'next';
type DestinationActiveProfile = 'first' | 'last' | 'none';

const runColumnVisibilityTransition = (update: () => void) => {
	if (
		typeof document.startViewTransition !== 'function' ||
		window.matchMedia('(prefers-reduced-motion: reduce)').matches
	) {
		update();
		return;
	}

	document.documentElement.classList.add(COLUMN_VISIBILITY_TRANSITION_CLASS);
	const transition = document.startViewTransition(async () => {
		flushSync(update);

		// Grid measures its column widths in the next task. Wait for that measurement before the browser captures
		// the new layout.
		await new Promise<void>((resolve) => window.setTimeout(resolve));
	});
	activeColumnVisibilityTransition = transition;
	const finish = () => {
		if (activeColumnVisibilityTransition !== transition) {
			return;
		}
		activeColumnVisibilityTransition = undefined;
		document.documentElement.classList.remove(COLUMN_VISIBILITY_TRANSITION_CLASS);
	};
	void transition.finished.then(finish, finish);
};

const useProfiles = () => {
	const initialState = createProfilesSequenceState('', 0, DEFAULT_PROFILE_LIMIT);
	const [sequence, dispatch] = useReducer(profilesSequenceReducer, initialState);
	const [profilesProperties, setProfilesPropertiesState] = useState<ProfileProperty[]>([]);
	const [profileSchemaProperties, setProfileSchemaProperties] = useState<ProfileSchemaProperties>({});
	const [profileSchema, setProfileSchema] = useState<ObjectType>();
	const [profileSchemaSessionID, setProfileSchemaSessionID] = useState(0);
	const [appliedFilter, setAppliedFilter] = useState<Filter | null>(null);

	const { api, handleError, redirect, selectedWorkspace, warehouse, workspaces } = useContext(AppContext);
	const sequenceRef = useRef(sequence);
	const executionRef = useRef(0);
	const appliedFilterRef = useRef<Filter | null>(null);
	const requestRef = useRef<ProfilesRangeRequest>();
	const inFlightRef = useRef<Map<string, InFlightProfilesRequest>>(new Map());
	const abortControllersRef = useRef<Set<AbortController>>(new Set());
	const pendingNavigationRef = useRef(false);
	const gridReplacementGenerationRef = useRef(0);
	const gridReplacementControllerRef = useRef<AbortController>();

	const applySequenceAction = useCallback((action: ProfilesSequenceAction) => {
		sequenceRef.current = profilesSequenceReducer(sequenceRef.current, action);
		dispatch(action);
	}, []);

	const abortCurrentExecution = useCallback(() => {
		for (const controller of abortControllersRef.current) {
			controller.abort();
		}
		abortControllersRef.current.clear();
		inFlightRef.current.clear();
		pendingNavigationRef.current = false;
	}, []);

	const cancelGridReplacement = useCallback(() => {
		gridReplacementGenerationRef.current++;
		gridReplacementControllerRef.current?.abort();
		gridReplacementControllerRef.current = undefined;
	}, []);

	const loadProfilesRange = useCallback(
		(
			executionID: number,
			request: ProfilesRangeRequest,
			first: number,
			limit: number,
			background: boolean,
		): Promise<boolean> => {
			const key = `${executionID}:${first}:${limit}`;
			const inFlight = inFlightRef.current.get(key);
			if (inFlight != null) {
				if (!background && inFlight.background) {
					return inFlight.promise.catch(() => loadProfilesRange(executionID, request, first, limit, false));
				}
				return inFlight.promise;
			}

			const controller = new AbortController();
			abortControllersRef.current.add(controller);
			let promise: Promise<boolean>;
			promise = api.workspaces.profiles
				.find(request.projection, request.filter, '', true, first, limit, request.schema, controller.signal)
				.then((response: FindProfilesResponse) => {
					if (controller.signal.aborted || executionRef.current !== executionID) {
						return false;
					}
					applySequenceAction({
						type: 'acceptRange',
						executionID,
						first,
						profiles: response.profiles,
						total: response.total,
						hasNext: response.hasNext,
					});
					return true;
				})
				.catch((error) => {
					if (executionRef.current !== executionID || error?.name === 'AbortError') {
						return false;
					}
					if (error instanceof UnprocessableError && error.code === 'SchemaNotAligned') {
						applySequenceAction({ type: 'markSchemaNotAligned', executionID });
						return false;
					}
					throw error;
				})
				.finally(() => {
					abortControllersRef.current.delete(controller);
					if (inFlightRef.current.get(key)?.promise === promise) {
						inFlightRef.current.delete(key);
					}
				});
			inFlightRef.current.set(key, { background, promise });
			return promise;
		},
		[api, applySequenceAction],
	);

	const reportProfilesError = useCallback(
		(error: unknown) => {
			if (error instanceof NotFoundError) {
				redirect(UI_BASE_PATH);
				handleError('The workspace does not exist anymore');
				return;
			}
			if (error instanceof UnprocessableError && error.code === 'DataWarehouseFailed') {
				handleError('An error occurred with the data warehouse');
				return;
			}
			handleError(error instanceof Error ? error : String(error));
		},
		[handleError, redirect],
	);

	const startExecution = useCallback(
		(
			profileProperties: ProfileProperty[],
			schema: ObjectType,
			pageSize: number,
			requestedFilter = appliedFilterRef.current,
		) => {
			cancelGridReplacement();
			abortCurrentExecution();
			const executionID = ++executionRef.current;
			const { filter, queryKey, request } = createProfilesExecution(
				selectedWorkspace,
				workspaces.find((candidate) => candidate.id === selectedWorkspace)?.assignedRoles,
				profileProperties,
				schema,
				pageSize,
				requestedFilter,
			);
			appliedFilterRef.current = filter;
			setAppliedFilter(filter);
			requestRef.current = request;
			applySequenceAction({ type: 'reset', queryKey, executionID, pageSize, projection: request.projection });
			void loadProfilesRange(executionID, request, 0, pageSize * 2, false).catch((error) => {
				if (executionRef.current !== executionID) {
					return;
				}
				applySequenceAction({ type: 'initialFailed', executionID });
				reportProfilesError(error);
			});
		},
		[
			abortCurrentExecution,
			applySequenceAction,
			cancelGridReplacement,
			loadProfilesRange,
			reportProfilesError,
			selectedWorkspace,
			workspaces,
		],
	);

	const loadSchemaAndStartExecution = useCallback(async () => {
		cancelGridReplacement();
		abortCurrentExecution();
		const schemaExecutionID = ++executionRef.current;
		applySequenceAction({
			type: 'reset',
			queryKey: `schema:${selectedWorkspace}:${schemaExecutionID}`,
			executionID: schemaExecutionID,
			pageSize: sequenceRef.current.pageSize,
			projection: [],
		});

		let schema: ObjectType;
		try {
			schema = await api.workspaces.profileSchema();
		} catch (error) {
			if (executionRef.current === schemaExecutionID) {
				applySequenceAction({ type: 'initialFailed', executionID: schemaExecutionID });
				reportProfilesError(error);
			}
			return;
		}
		if (executionRef.current !== schemaExecutionID) {
			return;
		}

		const { properties, schemaProperties } = profileMetadata(schema);
		setProfileSchema(schema);
		setProfileSchemaSessionID((sessionID) => sessionID + 1);
		setProfileSchemaProperties(schemaProperties);
		setProfilesPropertiesState(properties);
		localStorage.setItem(PROFILES_PROPERTIES_KEY, JSON.stringify(properties));
		startExecution(properties, schema, sequenceRef.current.pageSize);
	}, [
		abortCurrentExecution,
		api,
		applySequenceAction,
		cancelGridReplacement,
		reportProfilesError,
		selectedWorkspace,
		startExecution,
	]);

	useEffect(() => {
		if (warehouse == null) {
			redirect('settings');
			handleError('Please connect to a data warehouse before proceeding');
			return;
		}
		appliedFilterRef.current = null;
		setAppliedFilter(null);
		void loadSchemaAndStartExecution();
		return () => {
			executionRef.current++;
			cancelGridReplacement();
			abortCurrentExecution();
		};
	}, [selectedWorkspace]);

	const prefetchAfterPage = useCallback(
		(first: number) => {
			const current = sequenceRef.current;
			const page = getProfilesPage(current, first);
			const nextFirst = first + current.pageSize;
			const request = requestRef.current;
			if (page == null || !page.hasNext || getProfilesPage(current, nextFirst) != null || request == null) {
				return;
			}
			void loadProfilesRange(current.executionID, request, nextFirst, current.pageSize, true).catch(() => {
				// Speculative failures are retried as foreground requests if the
				// user subsequently requests this boundary.
			});
		},
		[loadProfilesRange],
	);

	const transitionToPage = useCallback(
		async (first: number, destinationActiveProfile: DestinationActiveProfile): Promise<boolean> => {
			const current = sequenceRef.current;
			if (first === current.visibleFirst || first < 0 || pendingNavigationRef.current) {
				return false;
			}
			const request = requestRef.current;
			if (request == null || current.isInitialLoading) {
				return false;
			}

			pendingNavigationRef.current = true;
			applySequenceAction({ type: 'startBoundary', first });
			try {
				if (getProfilesPage(sequenceRef.current, first) == null) {
					const accepted = await loadProfilesRange(
						current.executionID,
						request,
						first,
						current.pageSize,
						false,
					);
					if (!accepted) {
						return false;
					}
				}
				if (executionRef.current !== current.executionID) {
					return false;
				}
				const page = getProfilesPage(sequenceRef.current, first);
				if (page == null) {
					return false;
				}
				let activeProfileID = '';
				if (destinationActiveProfile === 'first') {
					activeProfileID = page.profiles[0]?.kpid ?? '';
				} else if (destinationActiveProfile === 'last') {
					activeProfileID = page.profiles[page.profiles.length - 1]?.kpid ?? '';
				}
				applySequenceAction({ type: 'commitPage', first, activeProfileID });
				prefetchAfterPage(first);
				return true;
			} catch (error) {
				if (executionRef.current === current.executionID) {
					reportProfilesError(error);
				}
				return false;
			} finally {
				if (executionRef.current === current.executionID) {
					pendingNavigationRef.current = false;
					applySequenceAction({ type: 'unlockBoundary' });
				}
			}
		},
		[applySequenceAction, loadProfilesRange, prefetchAfterPage, reportProfilesError],
	);

	const goToProfilesPage = useCallback(
		(first: number) => {
			const current = sequenceRef.current;
			const hasActiveProfile = current.activeProfileID !== '';
			const activeAtDestination = !hasActiveProfile ? 'none' : first > current.visibleFirst ? 'first' : 'last';
			void transitionToPage(first, activeAtDestination);
		},
		[transitionToPage],
	);

	const navigateProfile = useCallback(
		(direction: ProfileNavigationDirection) => {
			const current = sequenceRef.current;
			if (pendingNavigationRef.current) {
				return;
			}
			const page = getProfilesPage(current);
			if (page == null || page.profiles.length === 0) {
				return;
			}
			const activeIndex = page.profiles.findIndex((profile) => profile.kpid === current.activeProfileID);
			if (activeIndex === -1) {
				const profile = direction === 'next' ? page.profiles[0] : page.profiles[page.profiles.length - 1];
				applySequenceAction({ type: 'setActiveProfile', profileID: profile.kpid });
				return;
			}
			const destinationIndex = activeIndex + (direction === 'next' ? 1 : -1);
			if (destinationIndex >= 0 && destinationIndex < page.profiles.length) {
				applySequenceAction({
					type: 'setActiveProfile',
					profileID: page.profiles[destinationIndex].kpid,
				});
				return;
			}
			if (direction === 'next' && page.hasNext) {
				void transitionToPage(current.visibleFirst + current.pageSize, 'first');
			} else if (direction === 'previous' && current.visibleFirst > 0) {
				void transitionToPage(Math.max(0, current.visibleFirst - current.pageSize), 'last');
			}
		},
		[applySequenceAction, transitionToPage],
	);

	const refreshProfiles = useCallback(() => {
		if (profileSchema != null) {
			startExecution(profilesProperties, profileSchema, sequenceRef.current.pageSize);
		}
	}, [profileSchema, profilesProperties, startExecution]);

	const resetProfilesSchema = useCallback(() => {
		appliedFilterRef.current = null;
		setAppliedFilter(null);
		void loadSchemaAndStartExecution();
	}, [loadSchemaAndStartExecution]);

	const reorderProfilesProperties = useCallback((properties: ProfileProperty[]) => {
		setProfilesPropertiesState(properties);
		localStorage.setItem(PROFILES_PROPERTIES_KEY, JSON.stringify(properties));
	}, []);

	const previewProfilesFilter = useCallback(
		async (filter: Filter | null, signal?: AbortSignal): Promise<CountProfilesResponse> => {
			const executionID = executionRef.current;
			try {
				return await api.workspaces.profiles.count(filter, profileSchema, signal);
			} catch (error) {
				if (!signal?.aborted && executionRef.current === executionID) {
					if (error instanceof UnprocessableError && error.code === 'SchemaNotAligned') {
						applySequenceAction({ type: 'markSchemaNotAligned', executionID });
					} else if (error instanceof NotFoundError) {
						reportProfilesError(error);
					}
				}
				throw error;
			}
		},
		[api, applySequenceAction, profileSchema, reportProfilesError],
	);

	const replaceProfilesExecution = useCallback(
		async ({
			activeProfileID,
			animateColumnVisibility,
			filter: requestedFilter,
			first,
			profileProperties,
			projection,
		}: ProfilesReplacement): Promise<boolean> => {
			if (profileSchema == null) {
				return false;
			}

			cancelGridReplacement();
			const generation = gridReplacementGenerationRef.current;
			const controller = new AbortController();
			gridReplacementControllerRef.current = controller;
			const pageSize = sequenceRef.current.pageSize;
			const { filter, queryKey, request } = createProfilesExecution(
				selectedWorkspace,
				workspaces.find((candidate) => candidate.id === selectedWorkspace)?.assignedRoles,
				profileProperties,
				profileSchema,
				pageSize,
				requestedFilter,
				projection,
			);

			try {
				const response = await api.workspaces.profiles.find(
					request.projection,
					request.filter,
					'',
					true,
					first,
					pageSize * 2,
					request.schema,
					controller.signal,
				);
				if (gridReplacementGenerationRef.current !== generation) {
					return false;
				}

				const commitReplacement = () => {
					if (gridReplacementGenerationRef.current !== generation) {
						return;
					}
					abortCurrentExecution();
					const executionID = ++executionRef.current;
					const retainedActiveProfileID = response.profiles
						.slice(0, pageSize)
						.some((profile) => profile.kpid === activeProfileID)
						? activeProfileID
						: '';
					appliedFilterRef.current = filter;
					requestRef.current = request;
					setAppliedFilter(filter);
					applySequenceAction({
						type: 'replaceExecution',
						queryKey,
						executionID,
						pageSize,
						projection: request.projection,
						first,
						activeProfileID: retainedActiveProfileID,
						profiles: response.profiles,
						total: response.total,
						hasNext: response.hasNext,
					});
				};

				if (animateColumnVisibility) {
					runColumnVisibilityTransition(commitReplacement);
				} else {
					commitReplacement();
				}
				return true;
			} catch (error) {
				if (gridReplacementGenerationRef.current !== generation || error?.name === 'AbortError') {
					return false;
				}
				if (error instanceof UnprocessableError && error.code === 'SchemaNotAligned') {
					applySequenceAction({ type: 'markSchemaNotAligned', executionID: sequenceRef.current.executionID });
				}
				throw error;
			} finally {
				if (gridReplacementControllerRef.current === controller) {
					gridReplacementControllerRef.current = undefined;
				}
			}
		},
		[
			abortCurrentExecution,
			api,
			applySequenceAction,
			cancelGridReplacement,
			profileSchema,
			selectedWorkspace,
			workspaces,
		],
	);

	const showProfilesFilter = useCallback(
		async (filter: Filter | null): Promise<boolean> => {
			try {
				return await replaceProfilesExecution({
					activeProfileID: '',
					filter,
					first: 0,
					profileProperties: profilesProperties,
				});
			} catch (error) {
				if (error instanceof NotFoundError) {
					reportProfilesError(error);
				}
				throw error;
			}
		},
		[profilesProperties, replaceProfilesExecution, reportProfilesError],
	);

	const setActiveProfileID = useCallback(
		(profileID: string) => applySequenceAction({ type: 'setActiveProfile', profileID }),
		[applySequenceAction],
	);

	const setProfilesPageSize = useCallback(
		(limit: number) => {
			if (limit === sequenceRef.current.pageSize || profileSchema == null) {
				return;
			}
			startExecution(profilesProperties, profileSchema, limit);
		},
		[profileSchema, profilesProperties, startExecution],
	);

	const updateProfilesProperties = useCallback(
		(properties: ProfileProperty[]) => {
			if (profileSchema == null) {
				return;
			}

			const current = sequenceRef.current;
			if (current.isInitialLoading || requestRef.current == null) {
				setProfilesPropertiesState(properties);
				localStorage.setItem(PROFILES_PROPERTIES_KEY, JSON.stringify(properties));
				startExecution(properties, profileSchema, current.pageSize);
				return;
			}

			const updateProperties = () => {
				setProfilesPropertiesState(properties);
				localStorage.setItem(PROFILES_PROPERTIES_KEY, JSON.stringify(properties));
			};

			const loadedRoots = new Set(current.projection);
			const visibleColumnsChanged = properties.some(
				(property, index) =>
					property.isUsed !== profilesProperties[index]?.isUsed &&
					loadedRoots.has(property.name.split('.')[0]),
			);
			if (visibleColumnsChanged) {
				runColumnVisibilityTransition(updateProperties);
			} else {
				updateProperties();
			}

			const assignedRoles = workspaces.find((candidate) => candidate.id === selectedWorkspace)?.assignedRoles;
			const requestedProjection = profileProjection(properties, assignedRoles);
			if (requestedProjection.every((root) => loadedRoots.has(root))) {
				cancelGridReplacement();
				return;
			}

			const projection = Array.from(new Set([...current.projection, ...requestedProjection])).sort();
			void replaceProfilesExecution({
				activeProfileID: current.activeProfileID,
				animateColumnVisibility: true,
				filter: appliedFilterRef.current,
				first: current.visibleFirst,
				profileProperties: properties,
				projection,
			}).catch((error) => {
				const availableRoots = new Set(sequenceRef.current.projection);
				setProfilesPropertiesState((currentProperties) => {
					let changed = false;
					const revertedProperties = currentProperties.map((property) => {
						if (!property.isUsed || availableRoots.has(property.name.split('.')[0])) {
							return property;
						}
						changed = true;
						return { ...property, isUsed: false };
					});
					if (changed) {
						localStorage.setItem(PROFILES_PROPERTIES_KEY, JSON.stringify(revertedProperties));
					}
					return changed ? revertedProperties : currentProperties;
				});
				reportProfilesError(error);
			});
		},
		[
			cancelGridReplacement,
			profileSchema,
			profilesProperties,
			replaceProfilesExecution,
			reportProfilesError,
			selectedWorkspace,
			startExecution,
			workspaces,
		],
	);

	const markProfileSchemaNotAligned = useCallback(
		() => applySequenceAction({ type: 'markSchemaNotAligned', executionID: sequenceRef.current.executionID }),
		[applySequenceAction],
	);

	const profiles = getVisibleProfiles(sequence);
	const visiblePage = getProfilesPage(sequence);
	const activeProfileIndex = profiles.findIndex((profile) => profile.kpid === sequence.activeProfileID);
	const canNavigatePreviousProfile = activeProfileIndex >= 0 && (activeProfileIndex > 0 || sequence.visibleFirst > 0);
	const canNavigateNextProfile =
		activeProfileIndex >= 0 && (activeProfileIndex < profiles.length - 1 || visiblePage?.hasNext === true);

	return {
		appliedFilter,
		previewProfilesFilter,
		showProfilesFilter,
		profiles,
		profilesTotal: sequence.total,
		profilesProperties,
		profileSchema,
		profileSchemaProperties,
		isLoading: sequence.isInitialLoading,
		isBoundaryLoading: sequence.pendingFirst != null,
		profilesFirst: sequence.visibleFirst,
		profilesLimit: sequence.pageSize,
		profilesProjection: sequence.projection,
		profilesQueryKey: sequence.queryKey,
		profilesExecutionID: sequence.executionID,
		profileSchemaSessionID,
		resetProfilesSchema,
		hasNextPage: visiblePage?.hasNext ?? false,
		activeProfileID: sequence.activeProfileID,
		canNavigateNextProfile,
		canNavigatePreviousProfile,
		isProfileSchemaNotAligned: sequence.schemaNotAligned,
		goToProfilesPage,
		markProfileSchemaNotAligned,
		navigateProfile,
		refreshProfiles,
		reorderProfilesProperties,
		setActiveProfileID,
		setProfilesPageSize,
		updateProfilesProperties,
	};
};

const profileMetadata = (schema: ObjectType) => {
	const storageProperties = localStorage.getItem(PROFILES_PROPERTIES_KEY);
	let preferences: ProfileProperty[] = [];
	if (storageProperties != null) {
		try {
			preferences = JSON.parse(storageProperties);
		} catch (error) {
			localStorage.removeItem(PROFILES_PROPERTIES_KEY);
		}
	}

	const flatSchema = flattenSchema(schema);
	const paths = Object.keys(flatSchema);
	const schemaProperties: ProfileSchemaProperties = {};
	for (const path of paths) {
		schemaProperties[path] = flatSchema[path].full;
	}

	const properties: ProfileProperty[] = [];
	for (const path of paths) {
		const isParent = paths.some((candidate) => candidate !== path && candidate.startsWith(`${path}.`));
		if (isParent) {
			continue;
		}
		const preference = preferences.find((property) => property.name === path);
		const isTypeChanged = preference != null && preference.type !== flatSchema[path].type;
		properties.push({
			label: getProfilePropertyPathLabel(path, schemaProperties),
			name: path,
			isUsed: preference != null && !isTypeChanged ? preference.isUsed : true,
			type: flatSchema[path].type,
		});
	}
	const preferencePositions = new Map(preferences.map((property, index) => [property.name, index]));
	properties.sort(
		(first, second) =>
			(preferencePositions.get(first.name) ?? preferences.length) -
			(preferencePositions.get(second.name) ?? preferences.length),
	);

	return { properties, schemaProperties };
};

const profileProjection = (
	properties: ProfileProperty[],
	assignedRoles: { firstName: string; lastName: string; email: string; country: string; photo: string } | undefined,
): string[] => {
	const roots = new Set<string>();
	for (const property of properties) {
		if (property.isUsed) {
			roots.add(property.name.split('.')[0]);
		}
	}
	if (assignedRoles != null) {
		for (const path of Object.values(assignedRoles)) {
			if (path !== '') {
				roots.add(path.split('.')[0]);
			}
		}
	}
	return Array.from(roots).sort();
};

const createProfilesExecution = (
	workspace: string,
	assignedRoles: { firstName: string; lastName: string; email: string; country: string; photo: string } | undefined,
	properties: ProfileProperty[],
	schema: ObjectType,
	pageSize: number,
	requestedFilter: Filter | null,
	requestedProjection?: string[],
) => {
	const filter = requestedFilter == null ? null : structuredClone(requestedFilter);
	const projection = requestedProjection ?? profileProjection(properties, assignedRoles);
	const identity: ProfilesQueryIdentity = {
		workspace,
		search: '',
		filter,
		order: [
			{ property: '_updated_at', desc: true },
			{ property: '_kpid', desc: true },
		],
		pageSize,
		projection,
		schemaFingerprint: createProfilesValueFingerprint(schema),
	};
	return {
		filter,
		queryKey: createProfilesQueryKey(identity),
		request: { filter, projection, schema },
	};
};

export { useProfiles };
export type { ProfileNavigationDirection };
