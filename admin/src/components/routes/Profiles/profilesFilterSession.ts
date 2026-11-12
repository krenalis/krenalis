import { Filter } from '../../../lib/api/types/pipeline';
import { CountProfilesResponse } from '../../../lib/api/types/responses';
import { getFilterConditions } from '../../../lib/core/pipeline';
import { formatString } from '../../../utils/filters';
import { createProfilesValueFingerprint } from './profilesSequence';

const MAX_FILTER_HISTORY = 50;

interface FilterHistoryEntry {
	filter: Filter | null;
	// Keep editor quoting separate from the parsed API values.
	draft: Filter | null;
	fingerprint: string;
}

interface PreviewCandidate {
	filter: Filter | null;
	fingerprint: string;
}

interface FilterPreview extends CountProfilesResponse {
	fingerprint: string;
}

interface FailedPreview {
	candidate: PreviewCandidate;
	message: string;
}

interface FilterSession {
	draft: Filter | null;
	history: FilterHistoryEntry[];
	historyIndex: number;
	preview?: FilterPreview;
	pending?: PreviewCandidate;
	failure?: FailedPreview;
}

const cloneProfileFilter = (filter: Filter | null): Filter | null => (filter == null ? null : structuredClone(filter));

const profileFilterFingerprint = (filter: Filter | null): string => createProfilesValueFingerprint(filter);

const profileFilterResultsFingerprint = (filter: Filter | null): string =>
	createProfilesValueFingerprint(filter == null ? null : normalizeSingleRuleGroupOperators(filter));

const normalizeSingleRuleGroupOperators = (filter: Filter): Filter => ({
	operator: filter.rules.length < 2 ? 'and' : filter.operator,
	rules: filter.rules.map((rule) => ('rules' in rule ? normalizeSingleRuleGroupOperators(rule) : rule)),
});

const createProfileFilterHistoryEntry = (filter: Filter | null, draft?: Filter | null): FilterHistoryEntry => {
	const editableFilter = cloneProfileFilter(draft === undefined ? filter : draft);
	if (draft === undefined && editableFilter != null) {
		for (const condition of getFilterConditions(editableFilter)) {
			if (condition.values != null) {
				condition.values = condition.values.map(formatString);
			}
		}
	}
	return {
		filter: cloneProfileFilter(filter),
		draft: editableFilter,
		fingerprint: profileFilterFingerprint(filter),
	};
};

const createProfilesFilterSession = (filter: Filter | null, total?: number): FilterSession => {
	const entry = createProfileFilterHistoryEntry(filter);
	return {
		draft: cloneProfileFilter(entry.draft),
		history: [entry],
		historyIndex: 0,
		preview: total == null ? undefined : { fingerprint: entry.fingerprint, total },
	};
};

const currentProfilesFilterHistoryEntry = (session: FilterSession): FilterHistoryEntry =>
	session.history[session.historyIndex];

const commitProfilesFilter = (session: FilterSession, filter: Filter | null, draft?: Filter | null): FilterSession => {
	const entry = createProfileFilterHistoryEntry(filter, draft);
	const current = currentProfilesFilterHistoryEntry(session);
	if (current.fingerprint === entry.fingerprint) {
		const draftFingerprint = profileFilterFingerprint(entry.draft);
		if (
			profileFilterFingerprint(current.draft) === draftFingerprint &&
			profileFilterFingerprint(session.draft) === draftFingerprint
		) {
			return session;
		}
		const history = [...session.history];
		history[session.historyIndex] = entry;
		return {
			...session,
			draft: cloneProfileFilter(entry.draft),
			history,
		};
	}

	let history = [...session.history.slice(0, session.historyIndex + 1), entry];
	if (history.length > MAX_FILTER_HISTORY) {
		history = history.slice(history.length - MAX_FILTER_HISTORY);
	}

	return {
		...session,
		draft: cloneProfileFilter(entry.draft),
		history,
		historyIndex: history.length - 1,
	};
};

export {
	cloneProfileFilter,
	commitProfilesFilter,
	createProfileFilterHistoryEntry,
	createProfilesFilterSession,
	currentProfilesFilterHistoryEntry,
	profileFilterFingerprint,
	profileFilterResultsFingerprint,
};
export type { FailedPreview, FilterHistoryEntry, FilterSession, PreviewCandidate };
