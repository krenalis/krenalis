import { ResponseProfile } from '../../../lib/api/types/responses';

interface ProfilesPage {
	first: number;
	profiles: ResponseProfile[];
	hasNext: boolean;
}

interface ProfilesQueryIdentity {
	workspace: string;
	search?: string;
	filter?: unknown;
	order: Array<{ property: string; desc: boolean }>;
	pageSize: number;
	projection: string[];
	schemaFingerprint: string;
}

interface ProfilesSequenceState {
	queryKey: string;
	executionID: number;
	projection: string[];
	pages: Record<number, ProfilesPage>;
	visibleFirst: number;
	pageSize: number;
	total: number;
	activeProfileID: string;
	isInitialLoading: boolean;
	pendingFirst?: number;
	schemaNotAligned: boolean;
}

type ProfilesSequenceAction =
	| {
			type: 'acceptRange';
			executionID: number;
			first: number;
			profiles: ResponseProfile[];
			total: number;
			hasNext: boolean;
	  }
	| {
			type: 'replaceExecution';
			queryKey: string;
			executionID: number;
			pageSize: number;
			projection: string[];
			first: number;
			activeProfileID: string;
			profiles: ResponseProfile[];
			total: number;
			hasNext: boolean;
	  }
	| { type: 'commitPage'; first: number; activeProfileID: string }
	| { type: 'initialFailed'; executionID: number }
	| { type: 'markSchemaNotAligned'; executionID?: number }
	| { type: 'reset'; queryKey: string; executionID: number; pageSize: number; projection: string[] }
	| { type: 'setActiveProfile'; profileID: string }
	| { type: 'startBoundary'; first: number }
	| { type: 'unlockBoundary' };

const createProfilesSequenceState = (
	queryKey: string,
	executionID: number,
	pageSize: number,
	projection: string[] = [],
): ProfilesSequenceState => ({
	queryKey,
	executionID,
	projection,
	pages: {},
	visibleFirst: 0,
	pageSize,
	total: 0,
	activeProfileID: '',
	isInitialLoading: true,
	schemaNotAligned: false,
});

const profilesSequenceReducer = (
	state: ProfilesSequenceState,
	action: ProfilesSequenceAction,
): ProfilesSequenceState => {
	switch (action.type) {
		case 'acceptRange': {
			if (action.executionID !== state.executionID) {
				return state;
			}

			const pages = addProfilesRange(state.pages, state.pageSize, action.first, action.profiles, action.hasNext);

			return {
				...state,
				pages: evictProfilesPages(pages, state.visibleFirst, state.pageSize),
				total: action.total,
				isInitialLoading: false,
			};
		}
		case 'replaceExecution': {
			const replacement = createProfilesSequenceState(
				action.queryKey,
				action.executionID,
				action.pageSize,
				action.projection,
			);
			return {
				...replacement,
				visibleFirst: action.first,
				activeProfileID: action.activeProfileID,
				pages: addProfilesRange({}, action.pageSize, action.first, action.profiles, action.hasNext),
				total: action.total,
				isInitialLoading: false,
			};
		}
		case 'commitPage':
			return {
				...state,
				visibleFirst: action.first,
				activeProfileID: action.activeProfileID,
				pages: evictProfilesPages(state.pages, action.first, state.pageSize),
				pendingFirst: undefined,
			};
		case 'initialFailed':
			return action.executionID === state.executionID ? { ...state, isInitialLoading: false } : state;
		case 'markSchemaNotAligned':
			return action.executionID == null || action.executionID === state.executionID
				? { ...state, schemaNotAligned: true, isInitialLoading: false }
				: state;
		case 'reset':
			return createProfilesSequenceState(action.queryKey, action.executionID, action.pageSize, action.projection);
		case 'setActiveProfile':
			return { ...state, activeProfileID: action.profileID };
		case 'startBoundary':
			return state.pendingFirst == null ? { ...state, pendingFirst: action.first } : state;
		case 'unlockBoundary':
			return { ...state, pendingFirst: undefined };
	}
};

const createProfilesQueryKey = (identity: ProfilesQueryIdentity): string => canonicalJSONStringify(identity);

const createProfilesValueFingerprint = (value: unknown): string => canonicalJSONStringify(value);

const getProfilesPage = (state: ProfilesSequenceState, first = state.visibleFirst): ProfilesPage | undefined =>
	state.pages[first];

const getVisibleProfiles = (state: ProfilesSequenceState): ResponseProfile[] => getProfilesPage(state)?.profiles ?? [];

const canonicalJSONStringify = (value: unknown): string => JSON.stringify(canonicalize(value));

const canonicalize = (value: unknown): unknown => {
	if (Array.isArray(value)) {
		return value.map(canonicalize);
	}
	if (value != null && typeof value === 'object') {
		const result: Record<string, unknown> = {};
		for (const key of Object.keys(value).sort()) {
			result[key] = canonicalize((value as Record<string, unknown>)[key]);
		}
		return result;
	}
	return value;
};

const addProfilesRange = (
	currentPages: Record<number, ProfilesPage>,
	pageSize: number,
	first: number,
	profiles: ResponseProfile[],
	hasNextAfterRange: boolean,
): Record<number, ProfilesPage> => {
	const pages = { ...currentPages };
	if (profiles.length === 0) {
		pages[first] = { first, profiles: [], hasNext: hasNextAfterRange };
		return pages;
	}
	for (let offset = 0; offset < profiles.length; offset += pageSize) {
		const pageFirst = first + offset;
		pages[pageFirst] = {
			first: pageFirst,
			profiles: profiles.slice(offset, offset + pageSize),
			hasNext: offset + pageSize < profiles.length || hasNextAfterRange,
		};
	}
	return pages;
};

const evictProfilesPages = (
	pages: Record<number, ProfilesPage>,
	visibleFirst: number,
	pageSize: number,
): Record<number, ProfilesPage> => {
	const retained: Record<number, ProfilesPage> = {};
	for (const page of Object.values(pages)) {
		if (Math.abs(page.first - visibleFirst) <= pageSize * 2) {
			retained[page.first] = page;
		}
	}
	return retained;
};

export {
	createProfilesQueryKey,
	createProfilesSequenceState,
	createProfilesValueFingerprint,
	getProfilesPage,
	getVisibleProfiles,
	profilesSequenceReducer,
};
export type { ProfilesPage, ProfilesQueryIdentity, ProfilesSequenceAction, ProfilesSequenceState };
