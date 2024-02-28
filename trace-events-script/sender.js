import { debug, getTime } from './utils.js'

const MaxBodySize = 500 * 1024
const MaxHiddenBodySize = 64 * 1024

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
	#debug
	#closed

	// constructor returns a new Sender that sends the events in the queue to
	// the provided endpoint using the provided write key. Events already in the
	// queue are promptly dispatched, while others will be sent as they are
	// added to the queue.
	constructor(writeKey, endpoint, queue) {
		this.#queue = queue
		this.#writeKey = JSON.stringify(writeKey)
		this.#endpoint = endpoint + 'batch'
		this.#post = this.#postFunc()
		if (!queue.isEmpty()) {
			this.#send()
		}
		this.#queueListener = () => {
			if (!this.#sending && this.#timeoutID == null) {
				this.#debug?.('events will be sent after', this.timeout, 'ms')
				this.#timeoutID = setTimeout(() => {
					this.#timeoutID = null
					this.#send()
				}, this.timeout)
			}
		}
		this.#queue.addEventListener(this.#queueListener)
	}

	// close closes the sender.
	close() {
		if (this.#timeoutID != null) {
			clearTimeout(this.#timeoutID)
		}
		this.#queue.removeEventListener(this.#queueListener)
		this.#closed = true
		this.#debug?.('sender closed')
	}

	// debug toggles debug mode.
	debug(on) {
		this.#debug = debug(on)
	}

	// Flush immediately flushes the events waiting to be sent.
	flush() {
		this.#send(true)
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
					this.#debug?.('sending', body.size, 'bytes using sendBeacon')
					if (!navigator.sendBeacon(endpoint, body)) {
						cb(new Error('User agent is unable to queue the data for transfer'))
						return
					}
					setTimeout(() => cb({ ok: true, status: 204, statusText: 'No Content' }))
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
						ok: res.ok,
						status: res.status,
						statusText: res.statusText,
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
					ok: 200 <= xhr.status && xhr.status <= 299,
					status: xhr.status,
					statusText: xhr.statusText,
				}
				cb(response)
			}
			xhr.send(body)
		}
	}

	// send sends the queued events. If flush is true, it sends a single request
	// within 64KB body size limit.
	//
	// send is invoked only when the queue becomes non-empty (flush is false) or
	// the flush function is called (flush is true). In all other scenarios,
	// once send is called, it is guaranteed to continue until the queue becomes
	// empty.
	#send(flush) {
		if (flush) {
			if (this.#queue.isEmpty() || this.#sending || !navigator.onLine) {
				return
			}
		} else {
			if (!navigator.onLine) {
				addEventListener('online', () => {
					this.#send()
				})
				return
			}
			const timeout = (this.#queue.age() + this.timeout) - getTime()
			if (timeout > 0) {
				this.#debug?.('events will be sent after', timeout, 'ms')
				this.#timeoutID = setTimeout(() => {
					this.#timeoutID = null
					this.#send()
				}, timeout)
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
		const maxSize = (flush ? MaxHiddenBodySize : MaxBodySize) - leading.length - trailing.size
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
		// Send the body. The 'text/plain' content type is required for Chrome starting from version 59 when using sendBeacon.
		const body = new Blob(parts, { type: 'text/plain' })
		const sent = events.length
		this.#debug?.('Sending', sent, 'events of', this.#queue.size(), '(', body.size, 'bytes )')
		try {
			this.#post(this.#endpoint, body, flush, (response) => {
				if (this.#closed) {
					return
				}
				this.#sending = false
				if (response instanceof Error) {
					if (flush) {
						this.#debug?.('cannot post events:', response)
						return
					}
					if (navigator.onLine) {
						const timeout = 1000
						if (this.#debug) {
							this.#debug('cannot post events, try again after', timeout, 'ms:', response)
						} else {
							console.warn(response.message)
						}
						this.#timeoutID = setTimeout(() => {
							this.#timeoutID = null
							this.#send()
						}, timeout)
					} else {
						this.#send()
					}
					return
				}
				if (!response.ok) {
					const timeout = 1000
					if (this.#debug) {
						this.#debug(
							`server responded with status ${response.status} ${response.statusText}, will retry after`,
							timeout,
							'ms',
						)
					} else {
						console.warn(`sending events, the server responded with status ${response.status} ${response.statusText}`)
					}
					this.#timeoutID = setTimeout(() => {
						this.#timeoutID = null
						this.#send()
					}, timeout)
					return
				}
				this.#debug?.(sent, 'events sent')
				this.#queue.remove(sent)
				const size = this.#queue.size()
				if (flush || size === 0) {
					return
				}
				this.#send()
			})
		} catch (error) {
			if (navigator.onLine) {
				console.warn(error.message)
			}
			this.#debug?.('cannot post events, try again after 100ms:', error)
			this.#timeoutID = setTimeout(() => {
				this.#timeoutID = null
				this.#send()
			}, 100)
		}
	}
}

export default Sender
export { MaxBodySize, MaxHiddenBodySize, Sender }
