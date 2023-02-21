// utm returns an object with the UTM parameters, or it returns undefined if there are none.
function utm() {
	let utm;
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
			utm = utm || {};
			utm[kv[0]] = v;
		} catch (_) {}
	}
	return utm;
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

export { utm, uuid };
