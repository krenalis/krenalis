import { debug, getTime, onVisibilityChange } from './utils.js';
import Queue from './queue.js';

const MaxBodySize = 500 * 1024;
const MaxHiddenBodySize = 64 * 1024;
const MaxEventSize = 32 * 1024;

class Sender {
	timeout = 300;
	#writeKey;
	#endpoint = '';
	#queue;
	#flushing = false;
	#post;
	#debug;

	constructor(writeKey, endpoint, debug) {
		this.#queue = new Queue(globalThis.localStorage, 'chichi_queue', MaxEventSize, debug);
		this.#writeKey = JSON.stringify(writeKey);
		this.#endpoint = endpoint;
		this.debug(debug);
		this.#post = this.#postFunc();
		onVisibilityChange((visible) => {
			if (!visible) {
				this.#flush(true);
			}
		});
		if (!this.#queue.isEmpty()) {
			setTimeout(() => {
				this.#flush();
			}, 20);
		}
	}

	send(event) {
		this.#debug?.(`received '${event.type}' event`, event);
		const wasEmpty = this.#queue.isEmpty();
		try {
			const bytes = this.#queue.append(event);
			if (bytes > MaxEventSize) {
				console.warn('event size (' + bytes + 'bytes) is greater then 32KB');
				return;
			}
		} catch (error) {
			if (error instanceof TypeError) {
				console.warn('cannot stringify the event to JSON:', error);
				return;
			}
			throw error;
		}
		if (wasEmpty) {
			this.#debug?.('events will be flushed after', this.timeout, 'ms');
			setTimeout(() => {
				this.#flush();
			}, this.timeout);
		}
	}

	// debug toggles debug mode.
	debug(on) {
		this.#queue.debug(on);
		this.#debug = debug(on);
	}

	// flush flushes the queued events. If hidden is true, it sends a single
	// request within 64KB body size limit.
	//
	// flush should be invoked only when the queue becomes non-empty (hidden is
	// false) or the page becomes hidden (hidden is true). In all other
	// scenarios, once flush is called, it is guaranteed to continue until the
	// queue becomes empty or the page becomes hidden.
	#flush(hidden) {
		if (hidden) {
			if (this.#queue.isEmpty() || this.#flushing || !navigator.onLine) {
				return;
			}
		} else {
			if (!navigator.onLine) {
				globalThis.addEventListener('online', () => {
					this.#flush();
				});
				return;
			}
			const timeout = (this.#queue.age() + this.timeout) - getTime();
			if (timeout > 0) {
				this.#debug?.('events will be flushed after', timeout, 'ms');
				setTimeout(() => {
					this.#flush();
				}, timeout);
				return;
			}
		}
		this.#flushing = true;
		const leading = '{"batch":[';
		const trailing = new Blob([
			'],"sentAt":"',
			new Date().toJSON(),
			'","writeKey":',
			this.#writeKey,
			'}',
		]);
		const maxSize = (hidden ? MaxHiddenBodySize : MaxBodySize) - leading.length - trailing.size;
		const events = this.#queue.read(maxSize, 1);
		const parts = [];
		parts.push(leading);
		for (let i = 0; i < events.length; i++) {
			if (i > 0) {
				parts.push(',');
			}
			parts.push(events[i]);
		}
		parts.push(trailing);
		// Send the body. The 'text/plain' content type is required for Chrome starting from version 59 when using sendBeacon.
		const body = new Blob(parts, { type: 'text/plain' });
		const sent = events.length;
		this.#debug?.('flushing', sent, 'events of', this.#queue.length(), '(', body.size, 'bytes )');
		try {
			this.#post(this.#endpoint, body, hidden, (response) => {
				this.#flushing = false;
				if (response instanceof Error) {
					if (hidden) {
						this.#debug?.('cannot post events:', response);
						return;
					}
					if (navigator.onLine) {
						const timeout = 1000;
						if (this.#debug) {
							this.#debug('cannot post events, try again after', timeout, 'ms:', response);
						} else {
							console.warn(response.message);
						}
						setTimeout(() => {
							this.#flush();
						}, timeout);
					} else {
						this.#flush();
					}
					return;
				}
				if (!response.ok) {
					const timeout = 1000;
					if (this.#debug) {
						this.#debug(
							`server responded with status ${response.status} ${response.statusText}, will retry after`,
							timeout,
							'ms',
						);
					} else {
						console.warn(`sending events, the server responded with status ${response.status} ${response.statusText}`);
					}
					setTimeout(() => {
						this.#flush();
					}, timeout);
					return;
				}
				this.#debug?.(sent, 'events sent');
				this.#queue.remove(sent);
				const length = this.#queue.length();
				if (hidden || length === 0) {
					return;
				}
				this.#flush();
			});
		} catch (error) {
			if (navigator.onLine) {
				console.warn(error.message);
			}
			this.#debug?.('cannot post events, try again after 100ms:', error);
			setTimeout(() => {
				this.#flush();
			}, 100);
		}
	}

	// postFunc returns a function that issues a POST to the specified endpoint
	// with the given body. If keepalive is true the request outlives the page.
	// It returns an object with properties 'ok', 'status' and 'statusText'.
	// Returns an Error value in case of error.
	#postFunc() {
		// ES5: "fetch" is not available.
		if (globalThis.fetch && typeof globalThis.fetch === 'function') {
			return (endpoint, body, keepalive, cb) => {
				// Firefox does not support the keepalive option with fetch, so use beacon if it is available.
				if (keepalive && typeof navigator.sendBeacon === 'function') {
					this.#debug?.('sending', body.size, 'bytes using sendBeacon');
					if (!navigator.sendBeacon(endpoint, body)) {
						cb(new Error('User agent is unable to queue the data for transfer'));
						return;
					}
					cb({ ok: true, status: 204, statusText: 'No Content' });
					return;
				}
				this.#debug?.('sending', body.size, 'bytes using fetch');
				const promise = fetch(endpoint, {
					method: 'POST',
					cache: 'no-cache',
					headers: {
						'Content-Type': 'text/plain',
					},
					redirect: 'error',
					body: body,
					keepalive: keepalive,
				});
				promise.then((res) => {
					const response = {
						ok: res.ok,
						status: res.status,
						statusText: res.statusText,
					};
					cb(response);
				}, cb);
			};
		}
		return (endpoint, body, _, cb) => {
			this.#debug?.('sending', body.size, 'bytes using XMLHttpRequest');
			const xhr = new XMLHttpRequest();
			xhr.open('POST', endpoint, true);
			xhr.setRequestHeader('Content-Type', 'text/plain');
			xhr.onerror = () => {
				cb(new Error('an error occurred processing the request'));
			};
			xhr.onreadystatechange = () => {
				if (xhr.readyState !== 4) {
					return;
				}
				const response = {
					ok: 200 <= xhr.status && xhr.status <= 299,
					status: xhr.status,
					statusText: xhr.statusText,
				};
				cb(response);
			};
			xhr.send(body);
		};
	}
}

export default Sender;
export { MaxBodySize, MaxEventSize, MaxHiddenBodySize, Sender };
