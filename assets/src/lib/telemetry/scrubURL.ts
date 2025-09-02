import { UI_BASE_PATH } from '../../constants/paths';
import { API_BASE_PATH } from '../api/api';

const ROUTE_SENSITIVE_QUERIES = ['code', 'authToken'];
const ROUTE_PATTERNS = [
	'/sign-up/!token',
	'/reset-password',
	'/reset-password/!token',
	'/workspaces',
	'/workspaces/create',
	'/connectors',
	'/connectors/:name',
	'/connectors/file/:name',
	'/connections/sources',
	'/connections/destinations',
	'/connections',
	'/connections/:id',
	'/connections/:id/actions',
	'/connections/:id/actions/edit/:action',
	'/connections/:id/actions/add/event/:eventType',
	'/connections/:id/actions/add/event',
	'/connections/:id/actions/add/:actionTarget',
	'/connections/:id/metrics',
	'/connections/:id/events',
	'/connections/:id/settings',
	'/connections/:id/identities',
	'/oauth/authorize',
	'/users',
	'/schema',
	'/schema/edit',
	'/settings',
	'/settings/general',
	'/settings/identity-resolution',
	'/settings/data-warehouse',
	'/organization',
	'/organization/members/current',
	'/organization/members/add',
	'/organization/members',
	'/organization/access-keys',
];

const REQUEST_SENSITIVE_QUERIES = ['authCode', 'redirectURI', 'filter'];
const REQUEST_PATTERNS = [
	'/actions/:id',
	'/connections/:id',
	'/connections/:id/event-write-keys/!key',
	'/connections/:src/links/:dst',
	'/events/listeners/:id',
	'/keys/!key',
	'/members/!id',
	'/workspaces/current',
	'/actions/errors/:start/:end',
	'/actions/executions',
	'/actions/executions/:id',
	'/actions/metrics/dates/:start/:end',
	'/actions/metrics/days/:days',
	'/actions/metrics/hours/:hours',
	'/actions/metrics/minutes/:minutes',
	'/connections',
	'/connections/:id/action-types',
	'/connections/:id/actions/schemas/:target',
	'/connections/:id/actions/schemas/Events',
	'/connections/:id/event-write-keys',
	'/connections/:id/files',
	'/connections/:id/files/absolute',
	'/connections/:id/files/sheets',
	'/connections/:id/schemas/event',
	'/connections/:id/schemas/user',
	'/connections/:id/tables',
	'/connections/:id/ui',
	'/connections/:id/users',
	'/connections/auth-token',
	'/connections/auth-url',
	'/connectors',
	'/connectors/:name',
	'/events',
	'/events/listeners/:id',
	'/events/schema',
	'/events/settings/!write-key',
	'/external-event-url',
	'/identity-resolution/latest',
	'/identity-resolution/settings',
	'/javascript-sdk-url',
	'/keys',
	'/members',
	'/members/current',
	'/members/invitations/!token',
	'/members/reset-password/!token',
	'/reporting/errors/enabled',
	'/skip-member-email-verification',
	'/system/transformations/languages',
	'/users',
	'/users/schema',
	'/users/schema/latest-alter',
	'/users/schema/suitable-as-identifiers',
	'/users/!id/events',
	'/users/!id/identities',
	'/users/!id/traits',
	'/warehouse',
	'/warehouse/types',
	'/workspaces',
	'/workspaces/current',
	'/actions',
	'/actions/:id/exec',
	'/actions/:id/ui-event',
	'/connections/:id/identities',
	'/connections/:id/preview-send-event',
	'/connections/:id/query',
	'/connections/:id/ui-event',
	'/events/listeners',
	'/events/:type',
	'/expressions-properties',
	'/identity-resolution/start',
	'/members/invitations',
	'/members/login',
	'/members/logout',
	'/transformations',
	'/ui',
	'/ui-event',
	'/validate-expression',
	'/warehouse/repair',
	'/workspaces',
	'/workspaces/test',
	'/actions/:id',
	'/actions/:id/schedule',
	'/actions/:id/status',
	'/connections/:id',
	'/identity-resolution/settings',
	'/keys/!key',
	'/members/current',
	'/members/invitations/!token',
	'/members/reset-password',
	'/members/reset-password/!token',
	'/users/schema',
	'/users/schema/preview',
	'/warehouse/mode',
	'/warehouse/test',
];

const buildPatternsMap = (
	patterns: string[],
): Record<string, { pattern: string; placeholders: string[]; sensitiveData: string[] }> => {
	const output: Record<string, any> = {};

	patterns.forEach((pattern) => {
		const placeholders: string[] = [];
		const sensitiveData: string[] = [];

		const segments = pattern.split('/').filter(Boolean);

		const regex = segments
			.map((segment) => {
				if (segment.startsWith(':')) {
					placeholders.push(segment.slice(1));
					return `([^/]+)`;
				}
				if (segment.startsWith('!')) {
					sensitiveData.push(segment.slice(1));
					return `([^/]+)`;
				}
				return segment;
			})
			.join('/');

		const regexKey = `^/${regex}$`;
		const cleanPattern = pattern.replace(/!/g, ':');

		output[regexKey] = {
			pattern: cleanPattern,
			placeholders,
			sensitiveData,
		};
	});

	return output;
};

const scrubURL = (url: string, isRequest: boolean): [string, Record<string, string>] => {
	const patternsMap = buildPatternsMap(isRequest ? REQUEST_PATTERNS : ROUTE_PATTERNS);

	let basePath: string;
	let sensitiveQueries: string[];
	if (isRequest) {
		basePath = API_BASE_PATH;
		sensitiveQueries = REQUEST_SENSITIVE_QUERIES;
	} else {
		basePath = UI_BASE_PATH.slice(0, UI_BASE_PATH.length - 1);
		sensitiveQueries = ROUTE_SENSITIVE_QUERIES;
	}

	const urlObj = new URL(url);
	const path = urlObj.pathname.replace(basePath, '');
	const extras: Record<string, string> = {};

	const matched = Object.entries(patternsMap).find(([regex]) => new RegExp(regex).test(path));
	let scrubbedPath = path;
	if (matched) {
		const [regex, { pattern, placeholders, sensitiveData }] = matched;
		scrubbedPath = pattern;
		const match = path.match(new RegExp(regex));
		placeholders.forEach((placeholder, index) => {
			const value = match[index + 1]; // index 0 is the entire path.
			if (!sensitiveData.includes(placeholder)) {
				const key = isRequest ? `request[${placeholder}]` : placeholder;
				extras[key] = value;
			}
		});
	}

	const paramsList = Array.from(urlObj.searchParams.entries());
	let params = [];
	for (const [key, value] of paramsList) {
		if (sensitiveQueries.includes(key)) {
			params.push([key, '[REDACTED]']);
		} else {
			params.push([key, value]);
		}
	}
	const scrubbedParams = params.map(([key, value]) => `${key}=${value}`).join('&');

	let finalPath = scrubbedPath;
	if (scrubbedParams !== '') {
		finalPath = `${scrubbedPath}?${scrubbedParams}`;
	}

	return [`${urlObj.origin}${basePath}${finalPath}`, extras];
};

export { scrubURL };
