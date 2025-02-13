import { debug, getTime } from './utils.js'

const MaxBodySize = 500 * 1024
const MaxKeepAliveBodySize = 64 * 1024

// Sender sends events read from a Queue to a destination.
class Sender {
	timeout = 300
	#writeKey
	#endpoint
	#queue
	#sending = false
	#timeoutID = null
	#post
	#queueListener
	#onlineListener
	#debug
	#closed

	// constructor returns a new Sender that sends the events in the queue to
	// the provided endpoint using the provided write key. Events already in the
	// queue will be dispatched as soon as possible, while others will be sent
	// as they are added to the queue.
	constructor(writeKey, endpoint, queue) {
		this.#queue = queue
		this.#writeKey = JSON.stringify(writeKey)
		this.#endpoint = endpoint
		this.#post = this.#postFunc()
		if (!queue.isEmpty()) {
			this.#setTimeout(this.#send)
		}
		this.#queueListener = () => {
			if (!this.#sending && this.#timeoutID == null) {
				this.#debug?.('events will be sent after', this.timeout, 'ms')
				this.#setTimeout(this.#send, this.timeout)
			}
		}
		this.#queue.addEventListener(this.#queueListener)
	}

	// close closes the sender.
	close() {
		if (this.#timeoutID != null) {
			clearTimeout(this.#timeoutID)
		}
		if (this.#onlineListener != null) {
			removeEventListener('online', this.#onlineListener)
		}
		this.#queue.removeEventListener(this.#queueListener)
		this.#closed = true
		this.#debug?.('sender closed')
	}

	// debug toggles debug mode.
	debug(on) {
		this.#debug = debug(on)
	}

	// Flush flushes the events. It should be called when the browser is about
	// to unload the page.
	flush() {
		this.#send(true)
	}

	// postFunc returns a function that issues a POST to the specified endpoint
	// with the given body. If keepalive is true the request outlives the page.
	// It returns an object with properties 'status', 'statusText', and
	// 'retryAfter'. Returns an Error value in case of error.
	#postFunc() {
		// ES5: "fetch" is not available.
		if (globalThis.fetch && typeof globalThis.fetch === 'function') {
			return (endpoint, body, keepalive, cb) => {
				// Firefox does not support the keepalive option with fetch, so use beacon if it is available.
				if (keepalive && typeof navigator.sendBeacon === 'function') {
					this.#debug?.('sending', body.size, 'bytes using sendBeacon')
					if (!navigator.sendBeacon(endpoint, body)) {
						cb(new Error('User agent is unable to queue the data for transfer'))
						return
					}
					setTimeout(cb)
					return
				}
				this.#debug?.('sending', body.size, 'bytes using fetch')
				const promise = fetch(endpoint, {
					method: 'POST',
					cache: 'no-cache',
					headers: {
						'Content-Type': 'text/plain',
					},
					redirect: 'error',
					body: body,
					keepalive: keepalive,
				})
				promise.then((res) => {
					const response = {
						status: res.status,
						statusText: res.statusText,
						retryAfter: res.headers.get('Retry-After'),
					}
					cb(response)
				}, cb)
			}
		}
		return (endpoint, body, _, cb) => {
			this.#debug?.('sending', body.size, 'bytes using XMLHttpRequest')
			const xhr = new XMLHttpRequest()
			xhr.open('POST', endpoint, true)
			xhr.setRequestHeader('Content-Type', 'text/plain')
			xhr.onerror = () => {
				cb(new Error('an error occurred processing the request'))
			}
			xhr.onreadystatechange = () => {
				if (xhr.readyState !== 4) {
					return
				}
				const response = {
					status: xhr.status,
					statusText: xhr.statusText,
					retryAfter: xhr.getResponseHeader('Retry-After'),
				}
				cb(response)
			}
			xhr.send(body)
		}
	}

	// postRetry sends a POST request to the specified endpoint with the
	// provided body. In case of an error, if the request is retriable,
	// it automatically retries the request.
	//
	// The callback is invoked in the following cases:
	//    * upon successful completion of the request
	//    * if the request has been sent with sendBeacon
	//    * if there was an error and the request is not retriable
	//    * if the sender has been closed.
	// The first argument passed to the callback indicates whether the request
	// was successful. It is null if the request has been sent with sendBeacon.
	#postRetry(endpoint, body, keepalive, retries, cb) {
		const tryPost = (response) => {
			const asBeacon = response == null
			const status = asBeacon || response instanceof Error ? null : response.status
			const isSuccessful = asBeacon ? null : status === 200 || status === 201
			const isRetriable = status === null || status === 429 || status === 408 || status === 500 ||
				(status === 501 && response.retryAfter != null) || 502 <= status && status <= 599
			if (this.#debug != null && !isSuccessful && !asBeacon && !this.#closed) {
				this.#debug(
					'failed sending',
					body.size,
					`bytes: ${
						status == null ? response.message : `server responded with status ${status} ${response.statusText}`
					}`,
				)
			}
			if (isSuccessful || asBeacon || !isRetriable || this.#closed) {
				cb(isSuccessful)
				return
			}
			if (!navigator.onLine) {
				this.#debug?.('browser is offline, pause sending events')
				this.#onlineListener = () => {
					removeEventListener('online', this.#onlineListener)
					this.#onlineListener = null
					this.#debug?.('browser is online again, retry sending events')
					this.#postRetry(endpoint, body, keepalive, 0, cb)
				}
				addEventListener('online', this.#onlineListener)
				return
			}
			let delay
			const base = 100
			const cap = 5 * 1000
			if (status === 429 || status === 501 || status === 503) {
				// Set the delay to match the value returned by the server in the 'Retry-After' header.
				let retryAfter = parseInt(response.retryAfter)
				if (isNaN(retryAfter)) {
					try {
						const date = new Date(response.retryAfter)
						retryAfter = (date - getTime()) / 1000
					} catch {
						// the date is not valid.
					}
				}
				if (!isNaN(retryAfter)) {
					delay = Math.min(Math.floor(retryAfter * 1000 + (status === 503 ? Math.random() * base : 0)), cap)
				}
			}
			if (delay == null) {
				delay = Math.floor(Math.random() * base + Math.min(base * 2 ** retries, cap))
			}
			this.#debug?.('retry sending the events after a delay of', delay, 'ms')
			this.#setTimeout(() => {
				this.#postRetry(endpoint, body, keepalive, retries + 1, cb)
			}, delay)
		}
		try {
			this.#post(endpoint, body, keepalive, tryPost)
		} catch (error) {
			tryPost(error)
		}
	}

	// Send sends the queued events. When keepalive is true, it sends a single
	// request within a 64KB body size limit, utilizing either the sendBeacon
	// function or the fetch function with the keepalive option. If sendBeacon
	// is used, it retains the sent events in the queue, resending them a second
	// time, with keepalive set to false.
	//
	// Send is automatically invoked when the queue is not empty (keepalive is
	// false) or when the flush function is invoked (keepalive is true). Once
	// invoked, it continues sending events until the queue is emptied.
	#send(keepalive) {
		if (keepalive) {
			if (this.#queue.isEmpty() || (this.#sending && this.#timeoutID == null)) {
				return
			}
			if (this.#timeoutID != null) {
				clearTimeout(this.#timeoutID)
				this.#timeoutID = null
			}
		} else {
			const timeout = (this.#queue.age() + this.timeout) - getTime()
			if (timeout > 0) {
				this.#debug?.('events will be sent after', timeout, 'ms')
				this.#setTimeout(this.#send, timeout)
				return
			}
		}
		this.#sending = true
		const leading = '{"batch":['
		const trailing = new Blob([
			'],"sentAt":"',
			new Date().toJSON(),
			'","writeKey":',
			this.#writeKey,
			'}',
		])
		const maxSize = (keepalive ? MaxKeepAliveBodySize : MaxBodySize) - leading.length - trailing.size
		const events = this.#queue.read(maxSize, 1)
		const parts = []
		parts.push(leading)
		for (let i = 0; i < events.length; i++) {
			if (i > 0) {
				parts.push(',')
			}
			parts.push(events[i])
		}
		parts.push(trailing)
		// Starting from version 59, Chrome requires the 'text/plain' type when using sendBeacon.
		const body = new Blob(parts, { type: 'text/plain' })
		this.#debug?.('sending', events.length, 'events of', this.#queue.size(), '(', body.size, 'bytes )')
		this.#postRetry(this.#endpoint, body, keepalive, 0, (isSuccessful) => {
			if (this.#closed) {
				return
			}
			this.#sending = false
			// Sent events are only removed if sendBeacon has not been used.
			if (isSuccessful != null) {
				this.#queue.remove(events)
			}
			if (!this.#queue.isEmpty()) {
				// Continuing in the next tick is mandatory if sendBeacon has been used.
				this.#setTimeout(this.#send)
			}
		})
	}

	// setTimeout calls the global setTimeout method to execute a function once
	// the timer expires. It also sets the timeoutID property with the returned
	// value, enabling the timeout to be canceled if needed.
	#setTimeout(functionRef, delay) {
		this.#timeoutID = setTimeout(() => {
			this.#timeoutID = null
			functionRef.call(this)
		}, delay)
	}
}

export default Sender
export { MaxBodySize, MaxKeepAliveBodySize, Sender }
