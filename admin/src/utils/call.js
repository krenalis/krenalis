export default async function call(url, method, body) {
	let request = {
		method: method === '' || method == null ? 'GET' : method,
		headers: {
			'X-Workspace': 1,
		},
	};

	if (body) request.body = JSON.stringify(body);

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
				error = 'Internal Server Error';
				break;
			case 400:
				error = 'Bad Request';
				break;
			default:
				error = 'Unknown Server Error';
				break;
		}
		return [null, error];
	}

	let contentType = res.headers.get('content-type');
	if (!contentType || contentType.indexOf('application/json') === -1) {
		return [null, null];
	}

	let data;
	try {
		data = await res.json();
	} catch (err) {
		return [null, `error while parsing json response from ${url}: ${err.message}`];
	}

	if (data != null && typeof data === 'object' && 'Error' in data) {
		return [null, data.Error];
	}

	return [data, null];
}
