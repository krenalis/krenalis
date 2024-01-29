import { getTime } from './utils.js';

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

	constructor(writeKey, endpoint) {
		this.#writeKey = JSON.stringify(writeKey);
		this.#endpoint = endpoint;
		this.#post = postFunc();
		onUnload(() => {
			this.#flush(true);
		});
	}

	send(event) {
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

	// flush flushes the queued events. If unloading is true, it sends a single
	// request within 64KB body size limit.
	#flush(unloading) {
		if (unloading) {
			if (this.#events.length === 0 || this.#flushing || !navigator.onLine) {
				return;
			}
		} else if (!navigator.onLine) {
			setTimeout(this.#flush.bind(this), this.timeout);
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
			size += event.b.size;
			if (size >= maxBodySize) {
				break;
			}
			if (i > 0) {
				size++;
				parts.push(',');
			}
			parts.push(event.b);
		}
		const sent = parts.length / 2;
		parts.push(trailing);
		const body = new Blob(parts);
		try {
			this.#post(this.#endpoint, body, unloading, (response) => {
				this.#flushing = false;
				if (response instanceof Error) {
					if (navigator.onLine) {
						console.warn(response.message);
					}
				} else if (!response.ok) {
					console.warn(`sending events, the server responded with status ${response.status} ${response.statusText}`);
				} else {
					this.#events.splice(0, sent);
				}
				if (unloading || this.#events.length === 0) {
					return;
				}
				const timeout = getTime() - (this.#events[0].t + this.timeout);
				if (timeout <= 0) {
					this.#flush();
				} else {
					setTimeout(this.#flush.bind(this), timeout);
				}
			});
		} catch (error) {
			if (navigator.onLine) {
				console.warn(error.message);
			}
			setTimeout(this.#flush.bind(this), 100);
		}
	}
}

// onUnload calls cb when the page unloads.
const onUnload = function () {
	let unloaded = false;
	return function (cb) {
		window.addEventListener('visibilitychange', () => {
			if (unloaded === (document.visibilityState === 'hidden')) {
				return;
			}
			unloaded = !unloaded;
			if (unloaded) {
				cb();
			}
		});
		window.addEventListener('pagehide', () => {
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
	if (window.fetch && typeof window.fetch === 'function') {
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
	return function (endpoint, body, keepalive, cb) {
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
