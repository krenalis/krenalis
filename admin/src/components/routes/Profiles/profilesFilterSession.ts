import { Filter } from '../../../lib/api/types/pipeline';
import { CountProfilesResponse } from '../../../lib/api/types/responses';
import { createProfilesValueFingerprint } from './profilesSequence';

const MAX_FILTER_HISTORY = 50;

interface FilterHistoryEntry {
	filter: Filter | null;
	fingerprint: string;
	total?: number;
	datasetVersion?: string;
}

interface PreviewCandidate {
	filter: Filter | null;
	fingerprint: string;
	historyIndex?: number;
}

interface FailedPreview {
	candidate: PreviewCandidate;
	message: string;
}

interface FilterSession {
	draft: Filter | null;
	history: FilterHistoryEntry[];
	historyIndex: number;
	datasetVersion?: string;
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

const createProfileFilterHistoryEntry = (
	filter: Filter | null,
	total?: number,
	datasetVersion?: string,
): FilterHistoryEntry => ({
	filter: cloneProfileFilter(filter),
	fingerprint: profileFilterFingerprint(filter),
	total,
	datasetVersion,
});

const createProfilesFilterSession = (filter: Filter | null, total: number, datasetVersion?: string): FilterSession => ({
	draft: cloneProfileFilter(filter),
	history: [createProfileFilterHistoryEntry(filter, datasetVersion == null ? undefined : total, datasetVersion)],
	historyIndex: 0,
	datasetVersion,
});

const currentProfilesFilterHistoryEntry = (session: FilterSession): FilterHistoryEntry =>
	session.history[session.historyIndex];

const hasReusableProfilesFilterPreview = (
	entry: FilterHistoryEntry,
	fingerprint: string,
	datasetVersion: string | undefined,
): boolean =>
	entry.fingerprint === fingerprint &&
	entry.total != null &&
	entry.datasetVersion != null &&
	entry.datasetVersion === datasetVersion;

const commitProfilesFilterPreview = (
	session: FilterSession,
	candidate: PreviewCandidate,
	response: CountProfilesResponse,
): FilterSession => {
	const entry = createProfileFilterHistoryEntry(candidate.filter, response.total, response.datasetVersion);
	if (candidate.historyIndex != null) {
		const history = [...session.history];
		history[candidate.historyIndex] = entry;
		return {
			...session,
			draft: cloneProfileFilter(candidate.filter),
			history,
			historyIndex: candidate.historyIndex,
			datasetVersion: response.datasetVersion,
			pending: undefined,
			failure: undefined,
		};
	}

	if (currentProfilesFilterHistoryEntry(session).fingerprint === candidate.fingerprint) {
		const history = [...session.history];
		history[session.historyIndex] = entry;
		return {
			...session,
			draft: cloneProfileFilter(candidate.filter),
			history,
			datasetVersion: response.datasetVersion,
			pending: undefined,
			failure: undefined,
		};
	}

	let history = [...session.history.slice(0, session.historyIndex + 1), entry];
	if (history.length > MAX_FILTER_HISTORY) {
		history = history.slice(history.length - MAX_FILTER_HISTORY);
	}

	return {
		...session,
		draft: cloneProfileFilter(candidate.filter),
		history,
		historyIndex: history.length - 1,
		datasetVersion: response.datasetVersion,
		pending: undefined,
		failure: undefined,
	};
};

export {
	cloneProfileFilter,
	commitProfilesFilterPreview,
	createProfileFilterHistoryEntry,
	createProfilesFilterSession,
	currentProfilesFilterHistoryEntry,
	hasReusableProfilesFilterPreview,
	profileFilterFingerprint,
	profileFilterResultsFingerprint,
};
export type { FailedPreview, FilterHistoryEntry, FilterSession, PreviewCandidate };
