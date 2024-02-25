import { debug, getTime } from './utils.js'

// Queue is an in memory queue made persistent on a Storage.
class Queue {
	#storage
	#key
	#maxItemSize
	#items = []
	#times = []
	#sizes = []
	#inSync = true
	#timeoutID = null
	#debug

	// constructor initializes a new Queue using the provided Storage (such as
	// sessionStorage or localStorage), using the provided key, and with each
	// item limited to a maximum size in bytes specified by maxItemSize.
	constructor(storage, key, maxItemSize, debug) {
		this.#storage = storage
		this.#key = key
		this.#maxItemSize = maxItemSize
		this.debug(debug)
		this.#restore()
	}

	// age returns the time, in milliseconds, when the item in the head of the
	// queue was added. Returns null if the queue is empty.
	age() {
		return this.isEmpty() ? null : this.#times[0]
	}

	// append appends item to the queue and returns the size in bytes of the
	// JSON form of item. It raises an exception with type TypeError if an
	// error occurs calling JSON.stringify on item. If the JSON size of item
	// in bytes is greater than maxItemSize, it does nothing, apart returning
	// that size.
	append(item) {
		const time = getTime()
		item = JSON.stringify(item)
		const size = new Blob([item]).size
		if (size > this.#maxItemSize) {
			return size
		}
		this.#items.push(item)
		this.#times.push(time)
		this.#sizes.push(size)
		this.#inSync = false
		if (this.#timeoutID == null) {
			this.#timeoutID = setTimeout(() => {
				this.#timeoutID = null
				this.#makePersistent(200)
			}, 20)
		}
		this.#debug?.(
			'appended',
			size,
			`bytes value to the '${this.#key}' queue (`,
			this.#items.length,
			'events in queue )',
		)
		return size
	}

	// close closes the queue. It tries to preserve the queue in the
	// localStorage before returning. No other calls to the queue's method
	// should be made after a call the close method.
	close() {
		if (this.#timeoutID != null) {
			clearTimeout(this.#timeoutID)
			this.#timeoutID = null
		}
		this.#makePersistent()
		this.#debug?.(`'${this.#key}' queue closed`)
	}

	// debug toggles debug mode.
	debug(on) {
		this.#debug = debug(on)
	}

	// isEmpty reports whether the queue is empty.
	isEmpty() {
		return this.#items.length === 0
	}

	// length returns the total number of items currently in the queue.
	length() {
		return this.#items.length
	}

	// makePersistent makes the queue persistent in the localStorage. It can be
	// called, for example, when the page becomes not visible, to immediately
	// make it persistent.
	makePersistent() {
		if (this.#timeoutID != null) {
			clearTimeout(this.#timeoutID)
		}
		this.#makePersistent()
	}

	// read returns the items at the head of the queue, for a maximum of
	// maxBytes bytes. It considers separatorSize bytes in the total bytes as if
	// the returned items had a separator. If MaxBytes is null, there is no
	// limit in bytes. If separatorSize is null, there is no separator.
	read(maxBytes, separatorSize) {
		if (maxBytes == null && !separatorSize) {
			return [].concat(this.#items)
		}
		let n = 0
		let bytes = 0
		const length = this.#sizes.length
		for (let i = 0; i < length; i++) {
			if (i > 0) {
				bytes += separatorSize
			}
			bytes += this.#sizes[i]
			if (bytes > maxBytes) {
				break
			}
			n++
		}
		return this.#items.slice(0, n)
	}

	// remove removes n items from the queue. If there are fewer than n items,
	// or n is null, it removes all of them.
	remove(n) {
		if (n == null || n > this.#times.length) {
			n = this.#times.length
			if (n === 0) {
				this.#debug?.(`no events to remove from '${this.#key}' queue`)
				return
			}
		}
		this.#items.splice(0, n)
		this.#times.splice(0, n)
		this.#sizes.splice(0, n)
		this.#inSync = false
		this.#debug?.('removed', n, `items from the '${this.#key}' queue (`, this.#items.length, 'item still in queue )')
		if (this.#timeoutID != null) {
			clearTimeout(this.#timeoutID)
		}
		this.#makePersistent(200)
	}

	// makePersistent makes the queue persistent in the localStorage. It is
	// called by the public makePersistent method or when changes occur in the
	// queue and the queue is not currently being synced (when this.#inSync is
	// false). The delay parameter specifies the duration, in milliseconds, to
	// wait before attempting again in case of an error. If delay is null, no
	// retry will be made.
	#makePersistent(delay) {
		if (this.#inSync) {
			return
		}
		let text = ''
		if (this.#items.length > 0) {
			text = this.#items.join('\n') + '\n' + this.#times.join(' ') + '\n' + this.#sizes.join(' ')
		}
		let bytes
		if (this.#debug) {
			bytes = new Blob([text]).size
		}
		try {
			this.#storage.setItem(this.#key, text)
		} catch (error) {
			if (delay == null) {
				this.#debug?.('cannot make', bytes, `bytes of the '${this.#key}' queue persistent:`, error.message)
				return
			}
			delay = Math.min(2 * delay, 5000)
			this.#debug?.(
				'cannot make',
				bytes,
				`bytes of the '${this.#key}' queue persistent (will retry after`,
				delay,
				'ms):',
				error.message,
			)
			this.#timeoutID = setTimeout(() => {
				this.#timeoutID = null
				this.#makePersistent(delay)
			}, delay)
			return
		}
		this.#inSync = true
		this.#debug?.(
			`made '${this.#key}' queue persistent (`,
			this.#times.length,
			'items, with a size of',
			bytes,
			'bytes )',
		)
	}

	// restore restores the queue from localStorage. If any errors occur while
	// accessing localStorage it does nothing. It is only called by the
	// constructor.
	//
	// If the queue persisted in localStorage has been corrupted, restore
	// only ensures that no internal Queue data becomes corrupted, but it does
	// not guarantee the validity of the JSON items, nor does it ensure that
	// their sizes correspond to the original item sizes or that their
	// timestamps match the original ones.
	#restore() {
		let text
		try {
			text = this.#storage.getItem(this.#key)
		} catch (error) {
			this.#debug?.(`cannot restore the '${this.#key}' queue:`, error.message)
			return
		}
		if (text == null || text === '') {
			this.#debug?.(`no '${this.#key}' queue to restore`)
			return
		}
		try {
			const items = text.split('\n')
			const sizes = items.pop().split(' ')
			const times = items.pop().split(' ')
			if (sizes.length !== items.length || times.length !== items.length) {
				this.#debug?.(
					`cannot restore the '${this.#key}' queue, it is malformed:\n--begin-queue-------\n${text}\n--end-queue---------\n`,
				)
				return
			}
			let bytes = 0
			for (let i = 0; i < items.length; i++) {
				sizes[i] = Number(sizes[i])
				times[i] = Number(times[i])
				bytes += sizes[i]
			}
			this.#items = items
			this.#times = times
			this.#sizes = sizes
			this.#debug?.('restored', items.length, 'items (', bytes, `bytes ) from the '${this.#key}' queue`)
		} catch {
			this.#debug?.(
				`cannot restore the '${this.#key}' queue, it is malformed:\n--begin-queue-------\n${text}\n--end-queue---------\n`,
			)
		}
	}
}

export default Queue
