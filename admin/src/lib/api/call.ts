import { NotFoundError, UnavailableError, UnprocessableError, UnauthorizedError } from './errors';
import JSONbig from 'json-bigint';
import * as Sentry from '@sentry/react';
import { scrubURL } from '../telemetry/scrubURL';

const call = async (url: string, method: string, workspaceID?: number, body?: any, opt?: any) => {
	let headers = {
		'Content-Type': 'application/json',
	};
	if (workspaceID != null && workspaceID !== 0) {
		headers['Krenalis-Workspace'] = workspaceID;
	}
	const request: RequestInit = {
		method: method,
		headers: headers,
		...opt,
	};

	if (body !== undefined) {
		try {
			request.body = JSONbig.stringify(body);
		} catch (err) {
			throw new Error(`error while serializing request body for ${url}: ${err.message}`);
		}
	}

	let res: Response;
	try {
		res = await fetch(url, request);
	} catch (err) {
		if (err.name === 'AbortError') {
			throw err;
		}
		throw new Error(`error while fetching ${url}: ${err.message}`);
	}

	if (res.status !== 200) {
		switch (res.status) {
			case 500:
				throw new Error('Internal server error');
			case 400:
			case 401:
			case 404:
			case 422:
			case 503:
				let parsed = await res.json();
				let { code, message, cause } = parsed.error;
				if (res.status === 400) {
					Sentry.withScope((scope) => {
						scope.setExtra('type', 'Bad Request');
						const [scrubbedURL, extras] = scrubURL(url, true);
						scope.setExtra('requestURL', scrubbedURL);
						for (const k of Object.keys(extras)) {
							scope.setExtra(k, extras[k]);
						}
						Sentry.captureException(new Error(message));
					});
					console.error(
						`%c meergo: ${message}${cause ? ' | Cause: ' + cause : ''}`,
						'background:#dc362e;color:#dcdcdc',
					);
					throw new Error(
						'An error occurred in the application. Server responded with a Bad Request. Please contact the administrator.',
					);
				} else if (res.status === 401) {
					throw new UnauthorizedError();
				} else if (res.status === 404) {
					throw new NotFoundError(message);
				} else if (res.status === 422) {
					throw new UnprocessableError(code, message, cause);
				} else if (res.status === 503) {
					throw new UnavailableError(message, cause);
				}
				break;
			default:
				throw new Error('Unknown error');
		}
	}

	const contentType = res.headers.get('content-type');
	if (!contentType || contentType.indexOf('application/json') === -1) {
		return null;
	}

	let text: string;
	try {
		text = await res.text();
	} catch (err) {
		throw new Error(`error while decoding response from ${url}: ${err.message}`);
	}

	let data: any;
	try {
		data = JSONbig.parse(text);
	} catch (err) {
		throw new Error(`error while parsing json response from ${url}: ${err.message}`);
	}

	return data;
};

export default call;
