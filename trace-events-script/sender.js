import { uuid } from './utils';

class Sender {
	#source = '';
	#endpoint = '';
	#timeout = 300;
	#timeoutID;
	#flushing = false;
	#events = [];

	constructor(source, endpoint) {
		this.#source = source;
		this.#endpoint = endpoint;
		onUnload(() => {
			this.flush(true);
		});
	}

	send(event) {
		this.#events.push(JSON.stringify(event));
		if (this.#events.length === 1) {
			this.#timeoutID = setTimeout(() => {
				this.flush(false);
			}, this.#timeout);
		}
	}

	flush(keepalive) {
		if (this.#flushing || !window.navigator.onLine) {
			return;
		}
		this.#flushing = true;

		let messageID = uuid();
		let sentAt = new Date();
		let body = '{"messageId":"' + messageID + '","sentAt":"' + sentAt.toJSON() + '","batch":[';
		for (let i = 0; i < this.#events.length; i++) {
			if (i > 0) {
				body += ',';
			}
			body += this.#events[i];
		}
		body += ']}';

		try {
			post(this.#endpoint, this.#source, body, keepalive, (res) => {
				if (res instanceof Error) {
					console.warn('cannot send events: ' + res.message);
					return;
				}
				if (!res.ok) {
					console.warn(
						'sending events, the server responded with status ' + res.status + ' ' + res.statusText
					);
					return;
				}
				this.#events.length = 0;
				clearTimeout(this.#timeoutID);
				this.#flushing = false;
			});
		} catch (e) {
			console.warn('cannot send events: ' + e.message);
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
			if (unloaded) cb();
		});
		window.addEventListener('pagehide', () => {
			if (!unloaded) {
				unloaded = true;
				cb();
			}
		});
	};
};

// post issues a POST to the specified endpoint with the given body, source is
// the source ID. If keepalive is true the request outlives the page.
// It returns an object with properties 'ok', 'status' and 'statusText'.
// Returns an Error value in case of error.
const post = (function () {
	// Legacy: ie10 and ie11 do not support fetch.
	if (window.fetch && typeof window.fetch === 'function') {
		return function (endpoint, source, body, keepalive, cb) {
			const promise = fetch(endpoint, {
				method: 'POST',
				cache: 'no-cache',
				headers: {
					'Content-Type': 'application/json',
					Authorization: 'Basic ' + btoa(source + ':'),
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
	return function (endpoint, source, body, keepalive, cb) {
		const xhr = new XMLHttpRequest();
		xhr.open('POST', endpoint, true);
		xhr.setRequestHeader('Content-Type', 'application/json');
		xhr.setRequestHeader('Authorization', 'Basic ' + btoa(source + ':'));
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
})();

export default Sender;
