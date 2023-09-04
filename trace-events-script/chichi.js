import Analytics from './analytics.js';

// set DEBUG to true if you are running Chichi's server on localhost.
const DEBUG = true;
const undefined = void 0;

function main() {
	const analytics = window.chichianalytics;

	const a = new Analytics(analytics.key, analytics.url);
	const methods = [
		'alias',
		'endSession',
		'getSessionId',
		'group',
		'identify',
		'page',
		'ready',
		'reset',
		'screen',
		'setAnonymousId',
		'startSession',
		'track',
		'user',
	];
	for (let i = 0; i < methods.length; i++) {
		const method = methods[i];
		analytics[method] = a[method].bind(a);
	}

	for (let i = 0; i < analytics.length; i++) {
		let event = analytics[i];
		analytics[event[0]](...event[1]);
	}

	// empty the array.
	analytics.length = 0;

	window.chichianalytics = a;
}

main();
