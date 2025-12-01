import { UI_BASE_PATH } from '../../constants/paths';
import { API_BASE_PATH } from '../api/api';

const ROUTE_SENSITIVE_QUERIES = ['code', 'authToken'];
const ROUTE_PATTERNS = [
	'/connections',
	'/connections/:id',
	'/connections/:id/events',
	'/connections/:id/identities',
	'/connections/:id/metrics',
	'/connections/:id/pipelines',
	'/connections/:id/pipelines/add/:pipelineTarget',
	'/connections/:id/pipelines/add/event',
	'/connections/:id/pipelines/add/event/:eventType',
	'/connections/:id/pipelines/edit/:pipeline',
	'/connections/:id/settings',
	'/connections/destinations',
	'/connections/sources',
	'/connectors',
	'/connectors/:code',
	'/connectors/file/:code',
	'/oauth/authorize',
	'/organization',
	'/organization/access-keys',
	'/organization/members',
	'/organization/members/add',
	'/organization/members/current',
	'/profile-unification',
	'/profile-unification/profiles',
	'/profile-unification/rules',
	'/profile-unification/schema',
	'/profile-unification/schema/edit',
	'/reset-password',
	'/reset-password/!token',
	'/settings',
	'/settings/data-warehouse',
	'/settings/general',
	'/sign-up/!token',
	'/workspaces',
	'/workspaces/create',
];

// REQUEST_SENSITIVE_QUERIES lists request query keys whose values must be
//  redacted before reporting to Sentry.
const REQUEST_SENSITIVE_QUERIES = [
	'authCode',
	'cursor',
	'filter',
	'formatSettings',
	'name',
	'order',
	'path',
	'properties',
	'redirectURI',
	'schema',
	'sheet',
	'type',
];

// REQUEST_PATTERNS lists API endpoint patterns. Prefix a segment with `:` for
// regular placeholders, or with `!` when the value is sensitive and must stay
// out of Sentry. Sensitive segments get reported using their placeholder name
// (e.g. `/tokens/!key` → `/tokens/:key`) and skipped when populating extras.
//
// Keep this list aligned with the server map in `cmd/endpoints.go`.
const REQUEST_PATTERNS = [
	'/connections',
	'/connections/:id',
	'/connections/:id/event-write-keys',
	'/connections/:id/event-write-keys/!key',
	'/connections/:id/files',
	'/connections/:id/files/absolute',
	'/connections/:id/files/sheets',
	'/connections/:id/identities',
	'/connections/:id/pipeline-types',
	'/connections/:id/pipelines/schemas/:target',
	'/connections/:id/pipelines/schemas/Events',
	'/connections/:id/preview-send-event',
	'/connections/:id/query',
	'/connections/:id/schemas/event',
	'/connections/:id/schemas/user',
	'/connections/:id/tables',
	'/connections/:id/ui',
	'/connections/:id/ui-event',
	'/connections/:id/users',
	'/connections/:src/links/:dst',
	'/connections/auth-token',
	'/connections/auth-url',
	'/connectors',
	'/connectors/:code',
	'/events',
	'/events/:type',
	'/events/listeners',
	'/events/listeners/:id',
	'/events/schema',
	'/events/settings/!write-key',
	'/expressions-properties',
	'/identity-resolution/latest',
	'/identity-resolution/settings',
	'/identity-resolution/start',
	'/keys',
	'/keys/:key',
	'/members',
	'/members/:id',
	'/members/current',
	'/members/invitations',
	'/members/invitations/!token',
	'/members/login',
	'/members/logout',
	'/members/reset-password',
	'/members/reset-password/!token',
	'/pipelines',
	'/pipelines/:id',
	'/pipelines/:id/exec',
	'/pipelines/:id/schedule',
	'/pipelines/:id/status',
	'/pipelines/:id/ui-event',
	'/pipelines/errors/:start/:end',
	'/pipelines/executions',
	'/pipelines/executions/:id',
	'/pipelines/metrics/dates/:start/:end',
	'/pipelines/metrics/days/:days',
	'/pipelines/metrics/hours/:hours',
	'/pipelines/metrics/minutes/:minutes',
	'/profiles',
	'/profiles/!mpid/attributes',
	'/profiles/!mpid/events',
	'/profiles/!mpid/identities',
	'/profiles/schema',
	'/profiles/schema/latest-alter',
	'/profiles/schema/preview',
	'/profiles/schema/suitable-as-identifiers',
	'/system/transformations/languages',
	'/transformations',
	'/ui',
	'/ui-event',
	'/validate-expression',
	'/warehouse',
	'/warehouse/drivers',
	'/warehouse/mode',
	'/warehouse/repair',
	'/warehouse/test',
	'/workspaces',
	'/workspaces/current',
	'/workspaces/test',
	`/connections/:id/identities`,
	`/connections/:id`,
	`/connectors/:code/documentation`,
	`/public/metadata`,
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
