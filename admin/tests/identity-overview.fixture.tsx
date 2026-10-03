import React from 'react';
import ReactDOM from 'react-dom/client';
import '@shoelace-style/shoelace/dist/themes/light.css';
import API from '../src/lib/api/api';
import AppContext from '../src/context/AppContext';
import IdentityOverview from '../src/components/routes/IdentityOverview/IdentityOverview';

declare global {
	interface Window {
		identityOverviewScenario: 'historyError' | 'latestError' | 'removed' | 'singleDay' | 'singleDayZero';
		identityOverviewRequests: string[];
	}
}

const api = new API('https://example.test', 'workspace111');
let hasRefreshed = false;
window.identityOverviewRequests = [];
window.fetch = async (input) => {
	const url = String(input);
	window.identityOverviewRequests.push(url);
	if (url.includes('/metrics/identity-resolution/latest')) {
		return new Response('null', { headers: { 'Content-Type': 'application/json' } });
	}
	if (url.includes('/identity-resolution/runs')) {
		return new Response('{"runs":[]}', { headers: { 'Content-Type': 'application/json' } });
	}
	if (url.endsWith('/refresh')) {
		hasRefreshed = true;
		return new Response(null);
	}
	if (url.endsWith('/latest')) {
		if (window.identityOverviewScenario === 'latestError') return new Response(null, { status: 500 });
		let total = window.identityOverviewScenario === 'removed' ? (hasRefreshed ? 0 : 5) : 8;
		if (window.identityOverviewScenario === 'singleDay') total = 10;
		if (window.identityOverviewScenario === 'singleDayZero') total = 0;
		return new Response(
			JSON.stringify({
				observedAt: hasRefreshed ? '2026-08-03T12:00:00Z' : '2026-08-02T12:00:00Z',
				total,
				anonymous: 0,
				recognized: total,
				withoutProfile: 0,
				connections: hasRefreshed
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
	api,
	connections: [],
	selectedWorkspace: 'workspace111',
	setTitle: () => undefined,
} as unknown as React.ContextType<typeof AppContext>;

ReactDOM.createRoot(document.getElementById('root')!).render(
	<AppContext.Provider value={context}>
		<IdentityOverview />
	</AppContext.Provider>,
);
