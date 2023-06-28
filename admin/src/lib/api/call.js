import { NotFoundError, BadRequestError, UnprocessableError } from './errors';

const call = async (url, method, body) => {
	const request = {
		method: method,
		headers: {
			'X-Workspace': 1,
		},
	};

	if (body !== undefined) request.body = JSON.stringify(body);

	let res;
	try {
		res = await fetch(url, request);
	} catch (err) {
		return [null, `error while fetching ${url}: ${err.message}`];
	}

	if (res.status !== 200) {
		let error;
		switch (res.status) {
			case 500:
				error = 'Internal server error';
				break;
			case 400:
			case 404:
			case 422:
				let parsed = await res.json();
				let { code, message, cause } = parsed.error;
				if (res.status === 400) {
					error = new BadRequestError(message, cause);
				} else if (res.status === 404) {
					error = new NotFoundError(message);
				} else if (res.status === 422) {
					error = new UnprocessableError(code, message, cause);
				}
				break;
			default:
				error = 'Unknown error';
				break;
		}
		return [null, error];
	}

	const contentType = res.headers.get('content-type');
	if (!contentType || contentType.indexOf('application/json') === -1) {
		return [null, null];
	}

	let data;
	try {
		data = await res.json();
	} catch (err) {
		return [null, `error while parsing json response from ${url}: ${err.message}`];
	}

	return [data, null];
};

export default call;
