// campaign returns an object with the UTM parameters of the campaign, or it returns undefined if there are none.
function campaign() {
	let campaign;
	// ES5: "URLSearchParams" is not available.
	const search = globalThis.location.search.substring(1).replace(/\?/g, '&');
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
		} catch (_) {
			// nothing.
		}
	}
	return campaign;
}

// getTime returns the current UTC time in milliseconds from the epoch.
function getTime() {
	return new Date().getTime();
}

// isPlainObject reports whether obj is a plain object.
function isPlainObject(obj) {
	return typeof obj === 'object' && !Array.isArray(obj) && obj != null;
}

// debug returns a logging function for debug messages if 'on' is true;
// otherwise, it returns undefined.
function debug(on) {
	if (on) {
		return (...msg) => {
			console.log(`[${getTime()}]`, ...msg);
		};
	}
}

// _uuid_imp returns a function that returns random UUIDs or undefined if the
// browser is not supported.
function _uuid_imp() {
	let crypto = globalThis.crypto;
	if (crypto && typeof crypto.randomUUID === 'function') {
		return () => crypto.randomUUID();
	}
	// The following statement could be simplified to "crypto ||= globalThis.msCrypto",
	// but it hasn't been done because it wouldn't be testable.
	// Therefore, do not change it.
	if (!crypto || typeof crypto.getRandomValues !== 'function') {
		crypto = globalThis.msCrypto;
	}
	if (crypto && typeof crypto.getRandomValues === 'function') {
		return function () {
			// See https://stackoverflow.com/questions/105034/#2117523
			return '10000000-1000-4000-8000-100000000000'.replace(
				/[018]/g,
				(c) => (c ^ (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16),
			);
		};
	}
	const URL = globalThis.URL;
	if (URL && typeof URL.createObjectURL === 'function') {
		return function () {
			const url = URL.createObjectURL(new Blob());
			const uuid = url.toString();
			URL.revokeObjectURL(url);
			return uuid.split(/[:\/]/g).pop();
		};
	}
}

// uuid returns a random UUID.
// The uuid function is undefined for unsupported browsers.
const uuid = _uuid_imp();

export { _uuid_imp, campaign, debug, getTime, isPlainObject, uuid };
