import { debug, getTime, uuid } from './utils.js'

// Queue is an in-memory data structure that can be persisted to storage.
class Queue {
	#storage
	#key
	#maxItemSize
	#items = []
	#times = []
	#sizes = []
	#toBeSaved = false
	#timeoutID
	#eventListeners = new Set()
	#debug

	// constructor initializes a new Queue using the provided Storage (such as
	// sessionStorage or localStorage), using the provided key, and with each
	// item limited to a maximum size in bytes specified by maxItemSize.
	//
	// If the key contains an "*", the storage key is obtained by replacing the
	// last "*" with a newly generated UUID when the queue is saved, and after
	// it has been saved, the queue is cleared.
	constructor(storage, key, maxItemSize) {
		this.#storage = storage
		this.#key = key
		this.#maxItemSize = maxItemSize
	}

	// addEventListener adds a listener that will be called when one or more
	// items are added to the queue.
	addEventListener(listener) {
		this.#eventListeners.add(listener)
	}

	// age returns the time, in milliseconds, when the item in the head of the
	// queue was added. Returns null if the queue is empty.
	age() {
		return this.isEmpty() ? null : this.#times[0]
	}

	// append appends item to the queue. It raises an exception with type
	// TypeError if an error occurs calling JSON.stringify on item, and raises
	// an exception with type ItemTooLargeError if the JSON size of item in
	// bytes is greater than maxItemSize.
	append(item) {
		const time = getTime()
		item = JSON.stringify(item)
		const size = new Blob([item]).size
		if (size > this.#maxItemSize) {
			throw new ItemTooLargeError(size)
		}
		this.#items.push(item)
		this.#times.push(time)
		this.#sizes.push(size)
		this.#toBeSaved = true
		if (this.#timeoutID == null) {
			this.#timeoutID = setTimeout(() => {
				this.#timeoutID = null
				this.#save(200)
			}, 20)
		}
		this.#debug?.(
			'appended',
			size,
			`bytes value to the '${this.#key}' queue (`,
			this.#items.length,
			'events in queue )',
		)
		this.#dispatchEvent()
	}

	// clear clears the queue removing all the items.
	clear() {
		if (this.#items.length > 0) {
			this.#items.length = 0
			this.#times.length = 0
			this.#sizes.length = 0
			this.#toBeSaved = true
		}
	}

	// close closes the queue. It tries to save the queue in the localStorage
	// before returning. No other calls to the queue's method should be made
	// after a call the close method.
	close() {
		if (this.#timeoutID != null) {
			clearTimeout(this.#timeoutID)
		}
		this.#save()
		this.#debug?.(`queue closed: '${this.#key}'`)
	}

	// debug toggles debug mode.
	debug(on) {
		this.#debug = debug(on)
	}

	// isEmpty reports whether the queue is empty.
	isEmpty() {
		return this.#items.length === 0
	}

	// load loads and appends the items in the queue from localStorage with the
	// provided key. If any errors occur while accessing localStorage it does
	// nothing. It is only called by the constructor.
	//
	// If the queue persisted in localStorage has been corrupted, restore
	// only ensures that no internal Queue data becomes corrupted, but it does
	// not guarantee the validity of the JSON items, nor does it ensure that
	// their sizes correspond to the original item sizes or that their
	// timestamps match the original ones.
	load(key) {
		let text
		try {
			text = this.#storage.getItem(key)
		} catch (error) {
			this.#debug?.(`cannot load '${key}' queue:`, error.message)
			return
		}
		if (text == null || text === '') {
			this.#debug?.(`no '${key}' queue to load`)
			return
		}
		try {
			const items = text.split('\n')
			const sizes = items.pop().split(' ')
			const times = items.pop().split(' ')
			if (sizes.length !== items.length || times.length !== items.length) {
				throw null
			}
			let bytes = 0
			for (let i = 0; i < items.length; i++) {
				sizes[i] = Number(sizes[i])
				times[i] = Number(times[i])
				bytes += sizes[i]
			}
			if (this.isEmpty()) {
				this.#items = items
				this.#times = times
				this.#sizes = sizes
			} else {
				let i = times.length - 1
				let j = this.#times.length - 1
				let k = times.length + this.#times.length - 1
				while (i >= 0 && j >= 0) {
					if (times[i] > this.#times[j]) {
						this.#items[k] = items[i]
						this.#times[k] = times[i]
						this.#sizes[k] = sizes[i]
						i--
					} else {
						this.#items[k] = this.#items[j]
						this.#times[k] = this.#times[j]
						this.#sizes[k] = this.#sizes[j]
						j--
					}
					k--
				}
				while (i >= 0) {
					this.#items[k] = items[i]
					this.#times[k] = times[i]
					this.#sizes[k] = sizes[i]
					i--
					k--
				}
			}
			this.#toBeSaved = this.#key !== key
			this.#debug?.('loaded', items.length, 'items (', bytes, `bytes ) from the '${key}' queue`)
			this.#dispatchEvent()
		} catch {
			this.#debug?.(
				`cannot load the '${key}' queue, it is malformed:\n--begin-queue-------\n${text}\n--end-queue---------\n`,
			)
		}
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

	// remove removes the provided items from the queue. items should be an
	// array with the items as returned by the read method. Only existing items
	// are removed, starting from the head of the queue until no items remain to
	// be removed. If items is null or empty, it does nothing.
	remove(items) {
		if (items === null || items.length === 0) {
			return
		}
		let n = 0
		for (let i = 0; i < items.length; i++) {
			const j = this.#items.indexOf(items[i])
			if (j >= 0) {
				this.#items.splice(j, 1)
				this.#times.splice(j, 1)
				this.#sizes.splice(j, 1)
				n++
			}
		}
		this.#debug?.('removed', n, `items from the '${this.#key}' queue (`, this.#items.length, 'item still in queue )')
		if (n > 0) {
			this.#toBeSaved = true
		}
		if (this.#timeoutID != null) {
			clearTimeout(this.#timeoutID)
		}
		this.#save(200)
	}

	// removeEventListener removes a listener added with the addEventListener
	// method.
	removeEventListener(listener) {
		this.#eventListeners.delete(listener)
	}

	// save immediately saves the queue in the localStorage. If the queue's key
	// contains a "*", the storage key is obtained by replacing the last "*"
	// with a newly generated UUID, and after it has been saved, the queue is
	// cleared.
	save() {
		if (this.#timeoutID != null) {
			clearTimeout(this.#timeoutID)
		}
		this.#save()
	}

	// setKey sets the queue's key. key can be a string or a function to be
	// invoked when necessary to get the key.
	setKey(key) {
		if (this.#key !== key) {
			this.#key = key
			this.#toBeSaved = this.#items.length > 0
		}
	}

	// size returns the total number of items currently in the queue.
	size() {
		return this.#items.length
	}

	// dispatchEvent dispatches events to the listeners added with the
	// addEventListener method.
	#dispatchEvent() {
		for (const listener of this.#eventListeners) {
			setTimeout(listener)
		}
	}

	// save saves the queue in the localStorage. It is called by the public save
	// method or when changes occur in the queue and the queue is not currently
	// being synced (when this.#syncing is false). The delay parameter specifies
	// the duration, in milliseconds, to wait before attempting again in case of
	// an error. If delay is null, no retry will be made.
	#save(delay) {
		if (!this.#toBeSaved) {
			return
		}
		let text = ''
		if (this.#items.length > 0) {
			text = this.#items.join('\n') + '\n' + this.#times.join(' ') + '\n' + this.#sizes.join(' ')
		}
		let bytes = 0
		if (this.#debug) {
			for (let i = 0; i < this.#sizes.length; i++) {
				bytes += this.#sizes[i]
			}
		}
		let clear = false
		let key = this.#key
		const p = key.lastIndexOf('*')
		if (p >= 0) {
			clear = true
			key = key.slice(0, p) + uuid() + key.slice(p + 1)
		}
		try {
			this.#storage.setItem(key, text)
		} catch (error) {
			if (delay == null) {
				this.#debug?.(`cannot save '${key}' queue (`, bytes, 'bytes ) :', error.message)
				return
			}
			delay = Math.min(2 * delay, 5000)
			this.#debug?.(`cannot save '${key}' queue (`, bytes, 'bytes ), will retry after', delay, 'ms:', error.message)
			this.#timeoutID = setTimeout(() => {
				this.#timeoutID = null
				this.#save(delay)
			}, delay)
			return
		}
		if (clear) {
			this.#items.length = 0
			this.#times.length = 0
			this.#sizes.length = 0
		}
		this.#toBeSaved = false
		this.#debug?.(
			`saved '${key}' queue (`,
			this.#times.length,
			'items, with a size of',
			bytes,
			'bytes )',
		)
	}
}

// ItemTooLargeError represents the error that occurs when attempting to add an
// item to the queue that exceeds the permissible size limit.
class ItemTooLargeError extends Error {
	#size
	constructor(size) {
		super('The item is too large')
		this.#size = size
		this.name = this.constructor.name
	}
}

export default Queue
export { ItemTooLargeError, Queue }
