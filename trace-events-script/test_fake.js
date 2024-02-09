import { assert, assertEquals, AssertionError } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import * as uuid from 'https://deno.land/std@0.212.0/uuid/v4.ts';
import { MaxBodySize } from './sender.js';
import * as utils from './utils.js';

// Fetch implements a fake fetch.
class Fetch {
	#installTime;
	#writeKey;
	#endpoint;
	#events = [];
	#wait = null;
	#fetch;
	#originalFetch;
	#debug;

	constructor(writeKey, endpoint, debug) {
		this.#writeKey = writeKey;
		this.#endpoint = endpoint;
		this.#fetch = async (resource, options) => {
			assertEquals(resource, endpoint);
			const events = await parseRequest(this.#writeKey, this.#installTime, options);
			this.#events.push(...events);
			const min = this.#wait?.min;
			if (min != null && this.#events.length >= min) {
				const events = this.#events;
				const resolve = this.#wait.resolve;
				this.#events = [];
				this.#wait = null;
				this.#debug?.(`promise resolution is resolved: Fetch.events(${min})`);
				resolve(events);
			}
			const res = new Response('', {
				status: 200,
				statusText: 'OK',
				headers: new Headers({ 'content-type': 'text/plain' }),
			});
			return res;
		};
		this.#debug = utils.debug(debug);
	}

	events(min) {
		if (this.#installTime == null) {
			return new Promise((_, reject) => {
				reject(new Error('Fake fetch is not installed'));
			});
		}
		if (this.#wait != null) {
			return new Promise((_, reject) => {
				reject(new Error('events already called'));
			});
		}
		return new Promise((resolve) => {
			if (this.#events.length < min) {
				this.#wait = {
					min: min,
					resolve: resolve,
				};
				this.#debug?.(`promise resolution is pending:  Fetch.events(${min})`);
			} else {
				const events = this.#events;
				this.#events = [];
				this.#wait = null;
				resolve(events);
			}
		});
	}

	install() {
		if (this.#originalFetch != null) {
			throw new Error('Fake fetch is already installed');
		}
		this.#installTime = utils.getTime();
		this.#events = [];
		this.#wait = null;
		this.#originalFetch = globalThis.fetch;
		assert(this.#originalFetch != null);
		globalThis.fetch = this.#fetch;
	}

	restore() {
		if (this.#originalFetch == null) {
			throw new Error('Fake fetch is not installed');
		}
		globalThis.fetch = this.#originalFetch;
		this.#originalFetch = null;
		if (this.#events.length > 0) {
			throw new AssertionError(
				`Fake fetch has been restored; however, there are ${this.#events.length} unread events`,
			);
		}
	}
}

// SendBeacon implements a fake sendBeacon.
class SendBeacon {
	#installTime;
	#writeKey;
	#endpoint;
	#events = [];
	#wait = null;
	#sendBeacon;
	#debug;

	constructor(writeKey, endpoint, debug) {
		this.#writeKey = writeKey;
		this.#endpoint = endpoint;
		this.#sendBeacon = (url, data) => {
			assertEquals(url, endpoint);
			assert(data instanceof Blob);
			assertEquals(data.type, 'text/plain');
			parseRequest(this.#writeKey, this.#installTime, {
				method: 'POST',
				headers: { 'Content-Type': data.type },
				body: data,
				redirect: 'error',
			}).then((events) => {
				this.#events.push(...events);
				const min = this.#wait?.min;
				if (min != null && this.#events.length >= min) {
					const events = this.#events;
					const resolve = this.#wait.resolve;
					this.#events = [];
					this.#wait = null;
					this.#debug?.(`promise resolution is resolved: SendBeacon.events(${min})`);
					resolve(events);
				}
			});
			return true;
		};
		this.#debug = utils.debug(debug);
	}

	events(min) {
		if (this.#installTime == null) {
			return new Promise((_, reject) => {
				reject(new Error('Fake sendBeacon is not installed'));
			});
		}
		if (this.#wait != null) {
			return new Promise((_, reject) => {
				reject(new Error('events already called'));
			});
		}
		return new Promise((resolve) => {
			if (this.#events.length < min) {
				this.#wait = {
					min: min,
					resolve: resolve,
				};
				this.#debug?.(`promise resolution is pending:  SendBeacon.events(${min})`);
			} else {
				const events = this.#events;
				this.#events = [];
				this.#wait = null;
				resolve(events);
			}
		});
	}

	install() {
		if (this.#installTime != null) {
			throw new Error('Fake sendBeacon is already installed');
		}
		this.#installTime = utils.getTime();
		this.#events = [];
		this.#wait = null;
		globalThis.navigator.sendBeacon = this.#sendBeacon;
	}

	restore() {
		if (this.#installTime == null) {
			throw new Error('Fake sendBeacon is not installed');
		}
		this.#installTime = null;
		if (this.#events.length > 0) {
			throw new AssertionError(
				`Fake sendBeacon has been restored; however, there are ${this.#events.length} unread events`,
			);
		}
	}
}

// Navigator is a fake Navigator.
class Navigator {
	language = 'en-US';
	userAgent =
		'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36';
	onLine = true;
	#originalNavigator;
	install() {
		if (this.#originalNavigator != null) {
			throw new Error('Fake Navigator is already installed');
		}
		this.#originalNavigator = globalThis.navigator;
		delete (globalThis.navigator);
		globalThis.navigator = this;
	}
	restore() {
		if (this.#originalNavigator == null) {
			throw new Error('Fake Navigator is not installed');
		}
		delete (globalThis.navigator);
		globalThis.navigator = this.#originalNavigator;
		this.#originalNavigator = null;
	}
}

// Storage implements a fake storage that raises an exception at each method
// call.
class Storage {
	length = 0;

	key() {
		throw new Error('No storage available');
	}

	getItem() {
		throw new Error('No storage available');
	}

	setItem() {
		throw new Error('Quota exceeded');
	}

	removeItem() {
		throw new Error('No storage available');
	}

	clear() {
		throw new Error('No storage available');
	}
}

// XMLHttpRequest is a fake XMLHttpRequest.
class XMLHttpRequest {
	static #installTime;
	static #writeKey;
	static #endpoint;
	static #events;
	static #wait;
	static #debug;
	#method;
	#url;
	#headers = new Headers();
	onerror;
	onreadystatechange;
	readyState;
	status;
	statusText;

	open(method, endpoint, async) {
		assert(endpoint, XMLHttpRequest.#endpoint);
		assert(async);
		this.#method = method.toUpperCase();
		this.#url = endpoint;
	}

	setRequestHeader(name, value) {
		this.#headers.set(name.toLowerCase(), value);
	}

	send(body) {
		this.readyState = 4;
		this.status = 200;
		this.statusText = 'OK';
		parseRequest(XMLHttpRequest.#writeKey, XMLHttpRequest.#installTime, {
			method: this.#method,
			headers: this.#headers,
			body: body,
			redirect: 'error',
		}).then((events) => {
			XMLHttpRequest.#events.push(...events);
			const min = XMLHttpRequest.#wait?.min;
			if (min != null && XMLHttpRequest.#events.length >= min) {
				const events = XMLHttpRequest.#events;
				const resolve = XMLHttpRequest.#wait.resolve;
				XMLHttpRequest.#events = [];
				XMLHttpRequest.#wait = null;
				XMLHttpRequest.#debug?.(`promise resolution is resolved: XMLHttpRequest.events(${min})`);
				resolve(events);
			}
		}).catch((error) => {
			console.error(error);
		});
		if (typeof this.onreadystatechange === 'function') {
			this.onreadystatechange();
		}
	}

	static events(min) {
		if (XMLHttpRequest.#installTime == null) {
			return new Promise((_, reject) => {
				reject(new Error('Fake XMLHttpRequest is not installed'));
			});
		}
		if (XMLHttpRequest.#wait != null) {
			return new Promise((_, reject) => {
				reject(new Error('events already called'));
			});
		}
		return new Promise((resolve) => {
			if (XMLHttpRequest.#events.length < min) {
				XMLHttpRequest.#wait = {
					min: min,
					resolve: resolve,
				};
				XMLHttpRequest.#debug?.(`promise resolution is pending:  XMLHttpRequest.events(${min})`);
			} else {
				const events = XMLHttpRequest.#events;
				XMLHttpRequest.#events = [];
				XMLHttpRequest.#wait = null;
				resolve(events);
			}
		});
	}

	static install(writeKey, endpoint, debug) {
		if (XMLHttpRequest.#installTime != null) {
			throw new Error('Fake XMLHttpRequest is already installed');
		}
		XMLHttpRequest.#installTime = utils.getTime();
		XMLHttpRequest.#events = [];
		XMLHttpRequest.#wait = null;
		XMLHttpRequest.#writeKey = writeKey;
		XMLHttpRequest.#endpoint = endpoint;
		XMLHttpRequest.#debug = utils.debug(debug);
		globalThis.XMLHttpRequest = XMLHttpRequest;
	}

	static restore() {
		if (XMLHttpRequest.#installTime == null) {
			throw new Error('Fake XMLHttpRequest is not installed');
		}
		XMLHttpRequest.#installTime = null;
		delete (globalThis.XMLHttpRequest);
		if (this.#events.length > 0) {
			throw new AssertionError(
				`Fake fetch has been restored; however, there are ${this.#events.length} unread events`,
			);
		}
	}
}

// RandomUUID implements a fake crypto.randomUUID function.
class RandomUUID {
	#uuid;
	#originalRandomUUID;
	constructor(uuid) {
		this.#uuid = uuid;
	}
	install() {
		if (this.#originalRandomUUID != null) {
			throw new Error('Fake crypto.randomUUID is already installed');
		}
		this.#originalRandomUUID = crypto.randomUUID.bind(crypto);
		crypto.randomUUID = () => this.#uuid;
	}
	restore() {
		if (this.#originalRandomUUID == null) {
			throw new Error('Fake crypto.randomUUID is not installed');
		}
		crypto.randomUUID = this.#originalRandomUUID;
		this.#originalRandomUUID = null;
	}
}

// parseRequest parses a request to the fake fetch and XMLHttpRequest.send functions
async function parseRequest(writeKey, minTime, options) {
	const now = utils.getTime();

	assertEquals(options.method, 'POST');
	let headers = options.headers;
	if (!(options.headers instanceof Headers)) {
		headers = new Headers(options.headers);
	}
	assertEquals(Array.from(headers.keys()).length, 1);
	assertEquals(headers.get('content-type'), 'text/plain');
	assertEquals(options.redirect, 'error');
	assertEquals(Boolean(options.keepalive), false);
	assert(options.body instanceof Blob);
	if (options.body.size > MaxBodySize) {
		throw new AssertionError(`batch body size (${options.body.size}) is greater than ${MaxBodySize}`);
	}

	const body = JSON.parse(await options.body.text());
	assertEquals(typeof body.batch, 'object');
	assert(body.batch instanceof Array);
	assert(body.batch.length > 0);
	assertEquals(typeof body.sentAt, 'string');
	const sentAt = new Date(body.sentAt);
	assert(minTime <= sentAt && sentAt <= now);
	assertEquals(body.writeKey, writeKey);

	const events = [];
	for (let i = 0; i < body.batch.length; i++) {
		const event = body.batch[i];
		assertEquals(typeof event, 'object');
		assertEquals(typeof event.messageId, 'string');
		assert(uuid.validate(event.messageId));
		events.push(event);
	}

	return events;
}

export { Fetch, Navigator, RandomUUID, SendBeacon, Storage, XMLHttpRequest };
