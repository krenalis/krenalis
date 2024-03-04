const isIE = !globalThis.ActiveXObject && 'ActiveXObject' in globalThis

// campaign returns an object with the UTM parameters of the campaign, or it returns undefined if there are none.
function campaign() {
	let campaign
	// ES5: "URLSearchParams" is not available.
	const search = globalThis.location.search.substring(1).replace(/\?/g, '&')
	const params = search.split('&')
	for (let i = 0; i < params.length; i++) {
		const u = params[i].split('utm_')
		if (u[0] !== '' || u[1] === undefined) {
			continue
		}
		const kv = u[1].split('=')
		if (kv[0] === '' || kv[1] === undefined || kv[1] === '') {
			continue
		}
		try {
			// ES5: "replaceAll" is not available.
			const v = decodeURIComponent(kv[1].replace(/\+/g, ' '))
			campaign = campaign || {}
			const k = kv[0] === 'campaign' ? 'name' : kv[0]
			campaign[k] = v
		} catch {
			// nothing.
		}
	}
	return campaign
}

// debug returns a logging function for debug messages if 'on' is true;
// otherwise, it returns undefined.
function debug(on) {
	if (on) {
		if (isIE) {
			return (...msg) => {
				console.debug(`[${getTime()}] chichi:`, ...msg)
			}
		}
		return (...msg) => {
			console.debug('%c chichi ', 'background:#606060;color:#eee', `[${getTime()}]`, ...msg)
		}
	}
}

const textDecoder = typeof globalThis.TextDecoder === 'function' ? new globalThis.TextDecoder() : null
const textEncoder = typeof globalThis.TextEncoder === 'function' ? new globalThis.TextEncoder() : null

// decodeBase64 returns a string represented by the base64 encoded string s.
// If s is prefixed by _, the subsequent characters of s are interpreted as
// UTF-16 encoded characters (each represented by pairs of bytes), instead of
// UTF-8.
function decodeBase64(s) {
	if (s === '') {
		return ''
	}
	const utf16 = s[0] === '_'
	if (utf16) {
		s = s.slice(1)
	}
	const b = atob(s)
	const buf = new Uint8Array(b.length)
	for (let i = 0; i < buf.length; i++) {
		buf[i] = b.charCodeAt(i)
	}
	if (!utf16) {
		return textDecoder.decode(buf)
	}
	return String.fromCharCode.apply(null, new Uint16Array(buf.buffer))
}

// encodeBase64 returns the base64 encoding of the string src. If TextDecoder
// and TextEncoder are not supported, it returns the base64 encoding of the
// UTF-16 encoded content of src, instead of UTF-8, prefixed by _. If src is
// empty, it returns an empty string.
function encodeBase64(src) {
	if (src === '') {
		return ''
	}
	let s
	// The condition has not simplified to "textDecoder && textEncoder" to allow tests.
	if (globalThis.TextDecoder && globalThis.TextEncoder) {
		s = btoa(String.fromCodePoint.apply(null, textEncoder.encode(src)))
	} else {
		const b = new Uint16Array(src.length)
		for (let i = 0; i < b.length; i++) {
			b[i] = src.charCodeAt(i)
		}
		s = '_' + btoa(String.fromCharCode.apply(null, new Uint8Array(b.buffer)))
	}
	return s.replace(/=+$/, '')
}

// getTime returns the current UTC time in milliseconds from the epoch.
function getTime() {
	return new Date().getTime()
}

// isPlainObject reports whether obj is a plain object.
function isPlainObject(obj) {
	return typeof obj === 'object' && !Array.isArray(obj) && obj != null
}

// isURL reports whether url is a URL.
function isURL(url) {
	if (typeof url !== 'string' || !/^https?:\/\/\S+$/.test(url)) {
		return false
	}
	if (typeof globalThis.URL === 'function') {
		try {
			new URL(url)
		} catch {
			return false
		}
		return true
	}
	const a = document.createElement('a')
	a.href = url
	return a.href !== '' && a.hostname !== ''
}

// log returns a logging function for log error messages on the console.
function log(...msg) {
	if (isIE) {
		console.error('chichi:', ...msg)
		return
	}
	console.error('%c chichi ', 'background:#dc362e;color:#dcdcdc', ...msg)
}

// onVisibilityChange calls cb when the browser shows or hides the current page.
// It passed as argument a boolean indicating if the page is visible.
function onVisibilityChange(cb) {
	function isVisible() {
		const state = document.visibilityState || document.webkitvisibilitychange
		return state !== 'hidden'
	}
	let visible = isVisible()
	const change = () => {
		if (visible !== isVisible()) {
			visible = !visible
			cb(visible)
		}
	}
	// IE 11 do not support 'visibilitychange'.
	// In Safari before 14 'visibilitychange' does not work on globalThis but works on document.
	// In Safari before 14.5 'visibilitychange' does not fire on page hide, but 'pagehide' does.
	document.addEventListener('visibilitychange', change)
	addEventListener('pagehide', change)
	addEventListener('pageshow', change)
}

// _uuid_imp returns a function that returns random UUIDs or undefined if the
// browser is not supported.
function _uuid_imp() {
	let crypto = globalThis.crypto
	if (crypto && typeof crypto.randomUUID === 'function') {
		return () => crypto.randomUUID()
	}
	// The following statement could be simplified to "crypto ||= globalThis.msCrypto",
	// but it hasn't been done because it wouldn't be testable.
	// Therefore, do not change it.
	if (!crypto || typeof crypto.getRandomValues !== 'function') {
		crypto = globalThis.msCrypto
	}
	if (crypto && typeof crypto.getRandomValues === 'function') {
		return function () {
			// See https://stackoverflow.com/questions/105034/#2117523
			return '10000000-1000-4000-8000-100000000000'.replace(
				/[018]/g,
				(c) => (c ^ (crypto.getRandomValues(new Uint8Array(1))[0] & (15 >> (c / 4)))).toString(16),
			)
		}
	}
	const URL = globalThis.URL
	if (URL && typeof URL.createObjectURL === 'function') {
		return function () {
			const url = URL.createObjectURL(new Blob())
			const uuid = url.toString()
			URL.revokeObjectURL(url)
			return uuid.split(/[:\/]/g).pop()
		}
	}
}

// uuid returns a random UUID.
// The uuid function is undefined for unsupported browsers.
const uuid = _uuid_imp()

export {
	_uuid_imp,
	campaign,
	debug,
	decodeBase64,
	encodeBase64,
	getTime,
	isPlainObject,
	isURL,
	log,
	onVisibilityChange,
	uuid,
}
