// campaign returns an object with the UTM parameters of the campaign, or it returns undefined if there are none.
function campaign() {
	let campaign;
	// ES5: "URLSearchParams" is not available.
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
			// ES5: "replaceAll" is not available.
			const v = decodeURIComponent(kv[1].replace(/\+/g, ' '));
			campaign = campaign || {};
			const k = kv[0] === 'campaign' ? 'name' : kv[0];
			campaign[k] = v;
		} catch (_) {}
	}
	return campaign;
}

// uuid returns a random UUID.
// It is undefined for unsupported browsers.
const uuid = (function () {
	let crypto = window.crypto;
	if (crypto && typeof crypto.randomUUID === 'function') {
		return () => crypto.randomUUID();
	}
	crypto ||= window.msCrypto;
	if (crypto && typeof crypto.getRandomValues === 'function') {
		return function () {
			// See https://stackoverflow.com/questions/105034/#2117523
			return '10000000-1000-4000-8000-100000000000'.replace(/[018]/g, (c) =>
				(c ^ (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16),
			);
		};
	}
	const URL = window.URL;
	if (URL && typeof URL.createObjectURL === 'function') {
		return function () {
			const url = URL.createObjectURL(new Blob());
			const uuid = url.toString();
			URL.revokeObjectURL(url);
			return uuid.split(/[:\/]/g).pop();
		};
	}
})();

// typesOf returns a string representing the types of the elements of the array arr.
// If arr is not an array, it throws an error. If arr is empty, it returns an empty string.
// For example, if arr is ['a', 5], it returns 'string,number'.
function typesOf(arr) {
	return arr.map((v) => typeof v).join(',');
}

export { campaign, uuid, typesOf };
