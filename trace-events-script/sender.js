import { debug, getTime } from './utils.js';

const MaxBodySize = 500 * 1024;
const MaxUnloadingBodySize = 64 * 1024;
const MaxEventSize = 32 * 1024;

class Sender {
	timeout = 300;
	#writeKey;
	#endpoint = '';
	#flushing = false;
	#events = [];
	#post;
	#debug;

	constructor(writeKey, endpoint) {
		this.#writeKey = JSON.stringify(writeKey);
		this.#endpoint = endpoint;
		this.#post = postFunc();
		onUnload(() => {
			this.#flush(true);
		});
	}

	send(event) {
		this.#debug?.('send', event.type, 'event', event);
		const t = getTime();
		const b = new Blob([JSON.stringify(event)]);
		if (b.size > MaxEventSize) {
			console.warn('event size (' + b.size + ' bytes) is greater then 32KB');
			return;
		}
		if (this.#events.length === 0) {
			setTimeout(this.#flush.bind(this), this.timeout);
		}
		this.#events.push({ t, b });
	}

	// debug toggles debug mode.
	debug(on) {
		this.#debug = debug(on);
	}

	// flush flushes the queued events. If unloading is true, it sends a single
	// request within 64KB body size limit.
	//
	// flush should be invoked only when the queue becomes non-empty (unloading
	// is false) or the page is unloading (unloading is true). In all other
	// scenarios, once flush is called, it is guaranteed to continue until the
	// queue becomes empty or the page is unloading.
	#flush(unloading) {
		if (unloading) {
			if (this.#events.length === 0 || this.#flushing || !navigator.onLine) {
				return;
			}
		} else if (!navigator.onLine) {
			globalThis.addEventListener('online', () => {
				this.#flush();
			});
			return;
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
		let size = leading.length + trailing.size;
		const parts = [leading];
		const maxBodySize = unloading ? MaxUnloadingBodySize : MaxBodySize;
		const length = this.#events.length;
		for (let i = 0; i < length; i++) {
			const event = this.#events[i];
			if (size + event.b.size >= maxBodySize) {
				break;
			}
			size += event.b.size;
			if (i > 0) {
				size++;
				parts.push(',');
			}
			parts.push(event.b);
		}
		const sent = parts.length / 2;
		parts.push(trailing);
		const body = new Blob(parts);
		this.#debug?.('flush', sent, 'events of', this.#events.length, 'in queue (', size, 'bytes )');
		try {
			this.#post(this.#endpoint, body, unloading, (response) => {
				this.#flushing = false;
				if (response instanceof Error) {
					if (unloading) {
						return;
					}
					if (navigator.onLine) {
						const timeout = 1000;
						if (this.#debug) {
							this.#debug('error occurred sending the events (will retry after', timeout, 'ms):\n', response);
						} else {
							console.warn(response.message);
						}
						setTimeout(this.#flush.bind(this), timeout);
					} else {
						this.#flush();
					}
					return;
				}
				if (!response.ok) {
					const timeout = 1000;
					if (this.#debug) {
						this.#debug(
							`server responded with status ${response.status} ${response.statusText} (will retry after`,
							timeout,
							'ms)',
						);
					} else {
						console.warn(`sending events, the server responded with status ${response.status} ${response.statusText}`);
					}
					setTimeout(this.#flush.bind(this), timeout);
					return;
				}
				this.#events.splice(0, sent);
				this.#debug?.(sent, 'events sent (', this.#events.length, 'remains in queue )');
				if (unloading || this.#events.length === 0) {
					return;
				}
				const timeout = (this.#events[0].t + this.timeout) - getTime();
				if (timeout <= 0) {
					this.#debug?.('continue to flush now ( events in queue were supposed to be sent', -timeout, 'ms ago )');
					this.#flush();
				} else {
					this.#debug?.(`continue to flush after ${timeout}ms`);
					setTimeout(this.#flush.bind(this), timeout);
				}
			});
		} catch (error) {
			if (navigator.onLine) {
				console.warn(error.message);
			}
			this.#debug?.('cannot post events, try again after 100ms: ', error);
			setTimeout(this.#flush.bind(this), 100);
		}
	}
}

// onUnload calls cb when the page unloads.
const onUnload = function () {
	let unloaded = false;
	return function (cb) {
		globalThis.addEventListener('visibilitychange', () => {
			if (unloaded === (document.visibilityState === 'hidden')) {
				return;
			}
			unloaded = !unloaded;
			if (unloaded) {
				cb();
			}
		});
		globalThis.addEventListener('pagehide', () => {
			if (!unloaded) {
				unloaded = true;
				cb();
			}
		});
	};
};

// postFunc returns a function that issues a POST to the specified endpoint with
// the given body. If keepalive is true the request outlives the page.
// It returns an object with properties 'ok', 'status' and 'statusText'.
// Returns an Error value in case of error.
function postFunc() {
	// ES5: "fetch" is not available.
	if (globalThis.fetch && typeof globalThis.fetch === 'function') {
		return function (endpoint, body, keepalive, cb) {
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
	return function (endpoint, body, _, cb) {
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

export { MaxBodySize, MaxEventSize, MaxUnloadingBodySize, Sender };
