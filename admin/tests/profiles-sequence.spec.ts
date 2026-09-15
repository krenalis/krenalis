import { expect, test } from '@playwright/test';
import { ResponseProfile } from '../src/lib/api/types/responses';
import { Filter } from '../src/lib/api/types/pipeline';
import { ObjectType } from '../src/lib/api/types/types';
import { validateAndNormalizeFilter } from '../src/lib/core/pipeline';
import {
	createProfilesQueryKey,
	createProfilesSequenceState,
	getProfilesPage,
	getVisibleProfiles,
	profilesSequenceReducer,
} from '../src/components/routes/Profiles/profilesSequence';
import {
	commitProfilesFilterPreview,
	createProfilesFilterSession,
	currentProfilesFilterHistoryEntry,
	profileFilterFingerprint,
	profileFilterResultsFingerprint,
} from '../src/components/routes/Profiles/profilesFilterSession';
import {
	createProfilesPreviewTiming,
	recordProfilesPreviewLatency,
} from '../src/components/routes/Profiles/profilesPreviewTiming';

const profile = (index: number): ResponseProfile => ({
	kpid: `profile-${index}`,
	updatedAt: '2026-09-01T00:00:00Z',
	attributes: { index },
});

const filterSchema: ObjectType = {
	kind: 'object',
	properties: [
		{
			name: 'email',
			prefilled: '',
			role: 'Both',
			type: { kind: 'string' },
			createRequired: false,
			updateRequired: false,
			readOptional: true,
			nullable: false,
			description: '',
		},
	],
};

test(`Split an initial look-ahead response into visible and adjacent windows`, () => {
	let state = createProfilesSequenceState('query', 1, 2);
	state = profilesSequenceReducer(state, {
		type: 'acceptRange',
		executionID: 1,
		first: 0,
		profiles: [profile(1), profile(2), profile(3), profile(4)],
		total: 5,
		hasNext: true,
	});

	expect(getVisibleProfiles(state).map(({ kpid }) => kpid)).toEqual(['profile-1', 'profile-2']);
	expect(getProfilesPage(state, 2)?.profiles.map(({ kpid }) => kpid)).toEqual(['profile-3', 'profile-4']);
	expect(getProfilesPage(state, 0)?.hasNext).toBe(true);
	expect(getProfilesPage(state, 2)?.hasNext).toBe(true);
});

test(`Ignore obsolete executions and accept changing data in the current execution`, () => {
	let state = createProfilesSequenceState('query', 2, 2);
	state = profilesSequenceReducer(state, {
		type: 'acceptRange',
		executionID: 1,
		first: 0,
		profiles: [profile(1)],
		total: 1,
		hasNext: false,
	});
	expect(getVisibleProfiles(state)).toEqual([]);

	state = profilesSequenceReducer(state, {
		type: 'acceptRange',
		executionID: 2,
		first: 0,
		profiles: [profile(1), profile(2)],
		total: 3,
		hasNext: true,
	});
	state = profilesSequenceReducer(state, {
		type: 'acceptRange',
		executionID: 2,
		first: 2,
		profiles: [profile(3)],
		total: 3,
		hasNext: false,
	});

	expect(state.schemaNotAligned).toBe(false);
	expect(getProfilesPage(state, 2)?.profiles).toEqual([profile(3)]);
});

test(`Do not let an obsolete mismatch mark the current execution stale`, () => {
	let state = createProfilesSequenceState('query', 2, 2);
	state = profilesSequenceReducer(state, {
		type: 'acceptRange',
		executionID: 2,
		first: 0,
		profiles: [profile(1)],
		total: 1,
		hasNext: false,
	});
	state = profilesSequenceReducer(state, { type: 'markSchemaNotAligned', executionID: 1 });

	expect(state.schemaNotAligned).toBe(false);
});

test(`Retain useful adjacent pages and evict distant profile windows`, () => {
	let state = createProfilesSequenceState('query', 1, 2);
	for (let first = 0; first <= 8; first += 2) {
		state = profilesSequenceReducer(state, {
			type: 'acceptRange',
			executionID: 1,
			first,
			profiles: [profile(first + 1), profile(first + 2)],
			total: 10,
			hasNext: first < 8,
		});
		state = profilesSequenceReducer(state, { type: 'commitPage', first, activeProfileID: '' });
	}

	expect(getProfilesPage(state, 2)).toBeUndefined();
	expect(getProfilesPage(state, 4)).toBeDefined();
	expect(getProfilesPage(state, 6)).toBeDefined();
	expect(getProfilesPage(state, 8)).toBeDefined();
});

test(`Accept an empty page without requiring a data version`, () => {
	let state = createProfilesSequenceState('query', 1, 50);
	state = profilesSequenceReducer(state, {
		type: 'acceptRange',
		executionID: 1,
		first: 0,
		profiles: [],
		total: 0,
		hasNext: false,
	});
	expect(getVisibleProfiles(state)).toEqual([]);
	expect(getProfilesPage(state)?.hasNext).toBe(false);
});

test(`Replace a sequence atomically after a candidate query succeeds`, () => {
	let state = createProfilesSequenceState('old-query', 1, 2);
	state = profilesSequenceReducer(state, {
		type: 'acceptRange',
		executionID: 1,
		first: 0,
		profiles: [profile(1), profile(2)],
		total: 4,
		hasNext: true,
	});
	state = profilesSequenceReducer(state, { type: 'setActiveProfile', profileID: 'profile-2' });

	state = profilesSequenceReducer(state, {
		type: 'replaceExecution',
		queryKey: 'filtered-query',
		executionID: 2,
		pageSize: 2,
		projection: ['email'],
		first: 2,
		activeProfileID: 'profile-4',
		profiles: [profile(3), profile(4), profile(5)],
		total: 3,
		hasNext: false,
	});

	expect(state.queryKey).toBe('filtered-query');
	expect(state.executionID).toBe(2);
	expect(state.projection).toEqual(['email']);
	expect(state.visibleFirst).toBe(2);
	expect(state.activeProfileID).toBe('profile-4');
	expect(state.total).toBe(3);
	expect(state.isInitialLoading).toBe(false);
	expect(getVisibleProfiles(state).map(({ kpid }) => kpid)).toEqual(['profile-3', 'profile-4']);
	expect(getProfilesPage(state, 4)?.profiles.map(({ kpid }) => kpid)).toEqual(['profile-5']);
});

test(`Reset the query execution and preserve page state across a failed transition`, () => {
	let state = createProfilesSequenceState('old-query', 1, 2);
	state = profilesSequenceReducer(state, {
		type: 'acceptRange',
		executionID: 1,
		first: 0,
		profiles: [profile(1), profile(2)],
		total: 4,
		hasNext: true,
	});
	state = profilesSequenceReducer(state, { type: 'setActiveProfile', profileID: 'profile-2' });
	state = profilesSequenceReducer(state, { type: 'startBoundary', first: 2 });
	state = profilesSequenceReducer(state, { type: 'unlockBoundary' });

	expect(state.visibleFirst).toBe(0);
	expect(state.activeProfileID).toBe('profile-2');
	expect(state.pendingFirst).toBeUndefined();

	state = profilesSequenceReducer(state, {
		type: 'reset',
		queryKey: 'new-query',
		executionID: 2,
		pageSize: 3,
		projection: ['country'],
	});
	expect(state.queryKey).toBe('new-query');
	expect(state.executionID).toBe(2);
	expect(state.pageSize).toBe(3);
	expect(state.projection).toEqual(['country']);
	expect(state.activeProfileID).toBe('');
	expect(state.pages).toEqual({});
});

test(`Build deterministic query identities from canonical filters and projections`, () => {
	const first = createProfilesQueryKey({
		workspace: 'workspace',
		filter: { operator: 'and', rules: [{ value: 1, property: 'country' }] },
		order: [{ property: 'updatedAt', desc: true }],
		pageSize: 50,
		projection: ['country', 'email'],
		schemaFingerprint: 'schema',
	});
	const second = createProfilesQueryKey({
		workspace: 'workspace',
		filter: { rules: [{ property: 'country', value: 1 }], operator: 'and' },
		order: [{ desc: true, property: 'updatedAt' }],
		pageSize: 50,
		projection: ['country', 'email'],
		schemaFingerprint: 'schema',
	});

	expect(first).toBe(second);
});

test(`Normalize filter drafts without silently accepting partial conditions`, () => {
	const filter: Filter = {
		operator: 'and',
		rules: [
			{ property: '', operator: '', values: [''] },
			{ property: 'email', operator: 'contains', values: ['example.com'] },
		],
	};
	expect(validateAndNormalizeFilter(filter, filterSchema, 'Destination', 'User')).toEqual({
		operator: 'and',
		rules: [{ property: 'email', operator: 'contains', values: ['example.com'] }],
	});
	expect(() =>
		validateAndNormalizeFilter(
			{ operator: 'and', rules: [{ property: 'email', operator: 'contains', values: [''] }] },
			filterSchema,
			'Destination',
			'User',
		),
	).toThrow('cannot be empty');
	expect(
		validateAndNormalizeFilter(
			{ operator: 'and', rules: [{ property: 'email', operator: 'does not exist', values: [] }] },
			filterSchema,
			'Destination',
			'User',
		),
	).toEqual({ operator: 'and', rules: [{ property: 'email', operator: 'does not exist' }] });
});

test(`Initialize filter history from the displayed filter and last observed total`, () => {
	const filter: Filter = {
		operator: 'and',
		rules: [{ property: 'email', operator: 'contains', values: ['example.com'] }],
	};
	const session = createProfilesFilterSession(filter, 12);

	expect(session.history).toHaveLength(1);
	expect(currentProfilesFilterHistoryEntry(session)).toEqual({
		filter,
		fingerprint: profileFilterFingerprint(filter),
		total: 12,
	});
	expect(session.draft).not.toBe(filter);
});

test(`Ignore logical operators in groups that contain fewer than two rules when comparing profile results`, () => {
	const condition = { property: 'email', operator: 'contains' as const, values: ['example.com'] };
	const oneRuleAnd: Filter = { operator: 'and', rules: [condition] };
	const oneRuleOr: Filter = { operator: 'or', rules: [condition] };
	const twoRulesAnd: Filter = { operator: 'and', rules: [condition, condition] };
	const twoRulesOr: Filter = { operator: 'or', rules: [condition, condition] };
	const nestedAnd: Filter = { operator: 'and', rules: [oneRuleAnd, condition] };
	const nestedOr: Filter = { operator: 'and', rules: [oneRuleOr, condition] };

	expect(profileFilterResultsFingerprint(oneRuleAnd)).toBe(profileFilterResultsFingerprint(oneRuleOr));
	expect(profileFilterResultsFingerprint(nestedAnd)).toBe(profileFilterResultsFingerprint(nestedOr));
	expect(profileFilterResultsFingerprint(twoRulesAnd)).not.toBe(profileFilterResultsFingerprint(twoRulesOr));
});

test(`Add only successful previews to filter history and truncate a redo branch`, () => {
	const filterB: Filter = {
		operator: 'and',
		rules: [{ property: 'email', operator: 'contains', values: ['example.com'] }],
	};
	const filterC: Filter = {
		operator: 'and',
		rules: [{ property: 'email', operator: 'contains', values: ['example.org'] }],
	};
	const filterD: Filter = {
		operator: 'and',
		rules: [{ property: 'email', operator: 'contains', values: ['example.net'] }],
	};
	let session = createProfilesFilterSession(null, 20);
	session = commitProfilesFilterPreview(
		session,
		{ filter: filterB, fingerprint: profileFilterFingerprint(filterB) },
		{ total: 10 },
	);
	session = commitProfilesFilterPreview(
		session,
		{ filter: filterC, fingerprint: profileFilterFingerprint(filterC) },
		{ total: 5 },
	);

	session = { ...session, draft: filterB, historyIndex: 1 };
	session = commitProfilesFilterPreview(
		session,
		{ filter: filterD, fingerprint: profileFilterFingerprint(filterD) },
		{ total: 0 },
	);

	expect(session.history.map(({ filter }) => filter)).toEqual([null, filterB, filterD]);
	expect(session.historyIndex).toBe(2);
	expect(currentProfilesFilterHistoryEntry(session).total).toBe(0);
});

test(`Refresh history counts when data changes`, () => {
	const filter: Filter = {
		operator: 'and',
		rules: [{ property: 'email', operator: 'contains', values: ['example.com'] }],
	};
	let session = createProfilesFilterSession(filter, 12);
	session = commitProfilesFilterPreview(
		session,
		{ filter, fingerprint: profileFilterFingerprint(filter), historyIndex: 0 },
		{ total: 14 },
	);
	expect(currentProfilesFilterHistoryEntry(session).total).toBe(14);
});

test(`Adapt the text preview delay only after enough stable latency measurements`, () => {
	let timing = createProfilesPreviewTiming();
	timing = recordProfilesPreviewLatency(timing, 2000);
	timing = recordProfilesPreviewLatency(timing, 2000);
	expect(timing.delayMs).toBe(1000);

	timing = recordProfilesPreviewLatency(timing, 2000);
	expect(timing.delayMs).toBe(1000);
	timing = recordProfilesPreviewLatency(timing, 2000);
	expect(timing.delayMs).toBe(1500);

	timing = recordProfilesPreviewLatency(timing, 5000);
	timing = recordProfilesPreviewLatency(timing, 5000);
	timing = recordProfilesPreviewLatency(timing, 5000);
	expect(timing.delayMs).toBe(1500);
	timing = recordProfilesPreviewLatency(timing, 5000);
	expect(timing.delayMs).toBe(2500);
	expect(timing.latencySamples).toHaveLength(5);
});

test(`Use the true median when adapting from an even number of latency measurements`, () => {
	let timing = createProfilesPreviewTiming();
	timing = recordProfilesPreviewLatency(timing, 1000);
	timing = recordProfilesPreviewLatency(timing, 5000);
	timing = recordProfilesPreviewLatency(timing, 5000);
	timing = recordProfilesPreviewLatency(timing, 1000);

	expect(timing.delayMs).toBe(1000);
	expect(timing.delayCandidate).toBe(1500);
});

test(`Suspend and conservatively restore automatic text previews`, () => {
	let timing = createProfilesPreviewTiming();
	timing = recordProfilesPreviewLatency(timing, 9000);
	timing = recordProfilesPreviewLatency(timing, 9000);
	timing = recordProfilesPreviewLatency(timing, 9000);
	expect(timing.suspended).toBe(true);

	timing = recordProfilesPreviewLatency(timing, 3000);
	expect(timing.suspended).toBe(true);
	timing = recordProfilesPreviewLatency(timing, 3000);
	expect(timing.suspended).toBe(false);
	expect(timing.delayMs).toBe(2500);
});

test('A successful preview preserves placeholders in the editable draft', () => {
	const filter: Filter = {
		operator: 'and',
		rules: [{ property: 'email', operator: 'contains', values: ['example'] }],
	};
	const draft: Filter = { ...filter, rules: [...filter.rules, { property: '', operator: 'is', values: [] }] };
	const session = { ...createProfilesFilterSession(null, 10), draft };
	const next = commitProfilesFilterPreview(
		session,
		{ filter, fingerprint: profileFilterFingerprint(filter) },
		{ total: 3 },
	);
	expect(next.draft).toEqual(draft);
	expect(currentProfilesFilterHistoryEntry(next).filter).toEqual(filter);
});
