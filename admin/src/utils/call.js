export default async function call(url, value) {
	let body, res;
	if (value) body = JSON.stringify(value);
	try {
		res = body
			? await fetch(url, {
					method: 'POST',
					body: body,
					headers: {
						'X-Workspace': 1,
					},
			  })
			: await fetch(url, {
					headers: {
						'X-Workspace': 1,
					},
			  });
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
