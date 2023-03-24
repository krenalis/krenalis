// campaign returns an object with the UTM parameters of the campaign, or it returns undefined if there are none.
function campaign() {
	let campaign;
	// Legacy: ie10 and ie11 do not support URLSearchParams.
	const search = window.location.search.substring(1).replace(/\?/g, '&');
	const params = search.split('&');
	for (let i = 0; i < params.length; i++) {
		const u = params[i].split('utm_');
		if (u[0] !== '' || u[1] === undefined) {
			continue;
		}
		const kv = u[1].split('=');
		if (kv[0] === '' || kv[1] === undefined || kv[1] === '') {
			continue;
		}
		try {
			// Legacy: ie10 and ie11 do not support replaceAll.
			const v = decodeURIComponent(kv[1].replace(/\+/g, ' '));
			campaign = campaign || {};
			const k = kv[0] === 'campaign' ? 'name' : kv[0];
			campaign[k] = v;
		} catch (_) {}
	}
	return campaign;
}

// uuid returns a random UUID.
function uuid() {
	if (window.crypto && typeof window.crypto.randomUUID === 'function') {
		return window.crypto.randomUUID();
	}
	// Legacy: ie10 does not support crypto.getRandomValues.
	const url = URL.createObjectURL(new Blob());
	const uuid = url.toString();
	URL.revokeObjectURL(url);
	return uuid.split(/[:\/]/g).pop();
}

// typesOf returns a string representing the types of the elements of the array arr.
// If arr is not an array, it throws an error. If arr is empty, it returns an empty string.
// For example, if arr is ['a', 5], it returns 'string,number'.
function typesOf(arr) {
	return arr.map((v) => typeof v).join(',');
}

export { campaign, uuid, typesOf };
