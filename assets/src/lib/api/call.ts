import { NotFoundError, BadRequestError, UnavailableError, UnprocessableError, LoginRequiredError } from './errors';

const call = async (url: string, method: string, body?: any, opt?: any) => {
	const request: RequestInit = {
		method: method,
		...opt,
	};

	if (body !== undefined) request.body = JSON.stringify(body);

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
			case 404:
			case 422:
			case 503:
				let parsed = await res.json();
				let { code, message, cause } = parsed.error;
				if (res.status === 400) {
					throw new BadRequestError(message, cause);
				} else if (res.status === 404) {
					throw new NotFoundError(message);
				} else if (res.status === 422) {
					if (code === 'LoginRequired') {
						throw new LoginRequiredError();
					} else {
						throw new UnprocessableError(code, message, cause);
					}
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

	let data: any;
	try {
		data = await res.json();
	} catch (err) {
		throw new Error(`error while parsing json response from ${url}: ${err.message}`);
	}

	return data;
};

export default call;
