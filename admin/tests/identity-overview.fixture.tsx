import React from 'react';
import ReactDOM from 'react-dom/client';
import { MemoryRouter } from 'react-router-dom';
import '@shoelace-style/shoelace/dist/themes/light.css';
import API from '../src/lib/api/api';
import AppContext from '../src/context/AppContext';
import IdentityOverview from '../src/components/routes/IdentityOverview/IdentityOverview';
import { ResolutionEffectivenessChart } from '../src/components/routes/IdentityOverview/IdentityOverview.components';
import { ResolutionEffectivenessPoint } from '../src/components/routes/IdentityOverview/IdentityOverview.helpers';

declare global {
	interface Window {
		identityOverviewScenario:
			| 'atomic'
			| 'connectionTypes'
			| 'legendFilters'
			| 'historyError'
			| 'latestError'
			| 'removed'
			| 'singleDay'
			| 'singleDayZero'
			| 'resolution'
			| 'resolutionAtIdentity'
			| 'resolutionSingleDayZero'
			| 'resolutionUnauthorized';
		identityOverviewEffectivenessData?: ResolutionEffectivenessPoint[];
		identityOverviewRequests: string[];
		identityOverviewRevision?: number;
		identityOverviewFailure?: string;
		identityOverviewHold?: string;
		identityOverviewPending: (() => void)[];
		identityOverviewChangeWorkspace: (workspace: string) => void;
	}
}

window.identityOverviewRequests = [];
window.identityOverviewPending = [];
window.fetch = async (input) => {
	const url = String(input);
	const revision = window.identityOverviewRevision ?? 0;
	const failure = window.identityOverviewFailure;
	window.identityOverviewRequests.push(url);
	if (window.identityOverviewHold != null && url.includes(window.identityOverviewHold)) {
		await new Promise<void>((release) => window.identityOverviewPending.push(release));
	}
	if (failure != null && url.includes(failure)) return new Response(null, { status: 500 });
	const hasLegendFilters = window.identityOverviewScenario === 'legendFilters';
	if (window.identityOverviewScenario === 'atomic' || hasLegendFilters) {
		const day = revision === 0 ? '2026-08-02' : '2026-08-03';
		const total = hasLegendFilters ? 1000 : revision === 0 ? 8 : 18;
		const recognized = hasLegendFilters ? 10 : total;
		const anonymous = total - recognized;
		const profiles = hasLegendFilters ? 100 : revision === 0 ? 3 : 6;
		const profilesRecognized = hasLegendFilters ? 10 : profiles;
		const resolution = {
			observedAt: `${day}T11:00:00Z`,
			identities: hasLegendFilters
				? { total: 125, anonymous: 100, recognized: 25 }
				: { total: profiles * 2, anonymous: 0, recognized: profiles * 2 },
			profiles: { total: profiles, anonymous: profiles - profilesRecognized, recognized: profilesRecognized },
			composition: {
				one: hasLegendFilters ? 80 : 0,
				two: hasLegendFilters ? 15 : profiles,
				three: hasLegendFilters ? 5 : 0,
				fourToTen: 0,
				elevenToTwenty: 0,
				moreThanTwenty: 0,
			},
			identitiesPerProfile: hasLegendFilters ? 1.25 : 2,
			linkedIdentitiesRate: hasLegendFilters ? 0.2 : 1,
		};
		let data: unknown;
		if (url.includes('/metrics/identity-resolution/latest')) {
			data = resolution;
		} else if (url.includes('/metrics/identity-resolution/dates/')) {
			data = [
				{
					day: '2026-07-31',
					identities: resolution.identities.total,
					profiles,
					profilesAnonymous: profiles - profilesRecognized,
					profilesRecognized,
					identitiesPerProfile: resolution.identitiesPerProfile,
					linkedIdentitiesRate: resolution.linkedIdentitiesRate,
				},
			];
		} else if (url.includes('/identity-resolution/runs')) {
			data = {
				runs: [
					{
						id: `run${revision}`,
						status: 'successful',
						startTime: `${day}T10:00:00Z`,
						endTime: `${day}T11:00:00Z`,
						error: null,
					},
				],
			};
		} else if (url.endsWith('/latest')) {
			data = {
				observedAt: `${day}T12:00:00Z`,
				total,
				anonymous,
				recognized,
				withoutProfile: 0,
				connections: [{ connection: 'connection11', anonymous, recognized, withoutProfile: 0 }],
			};
		} else {
			data = [{ day: '2026-07-31', total: total - 2, anonymous, recognized: recognized - 2 }];
		}
		return new Response(JSON.stringify(data), { headers: { 'Content-Type': 'application/json' } });
	}
	if (url.includes('/metrics/identity-resolution/latest')) {
		if (window.identityOverviewScenario === 'resolutionUnauthorized') {
			return new Response('{"error":{}}', { status: 401, headers: { 'Content-Type': 'application/json' } });
		}
		if (
			window.identityOverviewScenario === 'resolution' ||
			window.identityOverviewScenario === 'resolutionAtIdentity' ||
			window.identityOverviewScenario === 'resolutionSingleDayZero'
		) {
			const hasZeroLinkRate = window.identityOverviewScenario === 'resolutionSingleDayZero';
			return new Response(
				JSON.stringify({
					observedAt:
						window.identityOverviewScenario === 'resolutionAtIdentity' || hasZeroLinkRate
							? '2026-08-02T12:00:00Z'
							: '2026-08-02T13:00:00Z',
					identities: { total: 10, anonymous: 0, recognized: 10 },
					profiles: { total: hasZeroLinkRate ? 10 : 5, anonymous: 0, recognized: hasZeroLinkRate ? 10 : 5 },
					composition: {
						one: hasZeroLinkRate ? 10 : 0,
						two: hasZeroLinkRate ? 0 : 5,
						three: 0,
						fourToTen: 0,
						elevenToTwenty: 0,
						moreThanTwenty: 0,
					},
					identitiesPerProfile: hasZeroLinkRate ? 1 : 2,
					linkedIdentitiesRate: hasZeroLinkRate ? 0 : 1,
				}),
				{ headers: { 'Content-Type': 'application/json' } },
			);
		}
		return new Response('null', { headers: { 'Content-Type': 'application/json' } });
	}
	if (url.includes('/metrics/identity-resolution/dates/')) {
		return new Response('[]', { headers: { 'Content-Type': 'application/json' } });
	}
	if (url.includes('/identity-resolution/runs')) {
		return new Response('{"runs":[]}', { headers: { 'Content-Type': 'application/json' } });
	}
	if (url.endsWith('/latest')) {
		if (window.identityOverviewScenario === 'latestError') return new Response(null, { status: 500 });
		if (window.identityOverviewScenario === 'connectionTypes') {
			const anonymous = revision === 1 ? 0 : 990;
			const hasRecognized = revision !== 2;
			return new Response(
				JSON.stringify({
					observedAt: revision === 0 ? '2026-08-02T12:00:00Z' : '2026-08-03T12:00:00Z',
					total: anonymous + (hasRecognized ? 30 : 0),
					anonymous,
					recognized: hasRecognized ? 30 : 0,
					withoutProfile: 0,
					connections: [
						{
							connection: 'connection11',
							anonymous,
							recognized: hasRecognized ? 10 : 0,
							withoutProfile: 0,
						},
						{
							connection: 'connection22',
							anonymous: 0,
							recognized: hasRecognized ? 20 : 0,
							withoutProfile: 0,
						},
					],
				}),
				{ headers: { 'Content-Type': 'application/json' } },
			);
		}
		const hasRemovedConnections = window.identityOverviewScenario === 'removed';
		let total = hasRemovedConnections ? 0 : 8;
		if (window.identityOverviewScenario === 'singleDay') total = 10;
		if (window.identityOverviewScenario === 'singleDayZero') total = 0;
		return new Response(
			JSON.stringify({
				observedAt: hasRemovedConnections ? '2026-08-03T12:00:00Z' : '2026-08-02T12:00:00Z',
				total,
				anonymous: 0,
				recognized: total,
				withoutProfile: 0,
				connections: hasRemovedConnections
					? []
					: [{ connection: 'connection11', anonymous: 0, recognized: total, withoutProfile: 0 }],
			}),
			{ headers: { 'Content-Type': 'application/json' } },
		);
	}
	if (window.identityOverviewScenario === 'historyError') return new Response(null, { status: 500 });
	if (window.identityOverviewScenario === 'singleDay' || window.identityOverviewScenario === 'singleDayZero') {
		return new Response('[]', { headers: { 'Content-Type': 'application/json' } });
	}
	return new Response(
		JSON.stringify([
			{ day: '2026-08-01', total: 5, anonymous: 0, recognized: 5 },
			{ day: '2026-08-02', total: 5, anonymous: 0, recognized: 5 },
		]),
		{ headers: { 'Content-Type': 'application/json' } },
	);
};

const context = {
	connections: [],
	selectedWorkspace: 'workspace111',
	setTitle: () => undefined,
} as unknown as React.ContextType<typeof AppContext>;

const root = ReactDOM.createRoot(document.getElementById('root')!);
window.identityOverviewChangeWorkspace = (workspace) => {
	root.render(
		<AppContext.Provider
			value={{ ...context, api: new API('https://example.test', workspace), selectedWorkspace: workspace }}
		>
			<MemoryRouter>
				<IdentityOverview />
			</MemoryRouter>
		</AppContext.Provider>,
	);
};
if (window.identityOverviewEffectivenessData != null) {
	root.render(<ResolutionEffectivenessChart data={window.identityOverviewEffectivenessData} loading={false} />);
} else {
	window.identityOverviewChangeWorkspace('workspace111');
}
