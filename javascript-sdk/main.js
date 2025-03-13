import { EndpointURLError, Meergo } from './meergo.js'
import { uuid } from './utils.js'

function main() {
	// Do nothing if the browser is not supported.
	if (!uuid) {
		return
	}

	const meergo = globalThis.meergo

	let e
	try {
		e = new Meergo(meergo.key, meergo.url, meergo.options)
	} catch (error) {
		if (error instanceof EndpointURLError) {
			console.error(error.message)
			return
		}
		throw error
	}

	void e.ready(function () {
		const methods = [
			'alias',
			'close',
			'debug',
			'endSession',
			'getAnonymousId',
			'getSessionId',
			'group',
			'identify',
			'page',
			'ready',
			'reset',
			'screen',
			'setAnonymousId',
			'startSession',
			'track',
			'user',
		]
		for (let i = 0; i < methods.length; i++) {
			const method = methods[i]
			meergo[method] = e[method].bind(e)
		}

		for (let i = 0; i < meergo.length; i++) {
			const event = meergo[i]
			meergo[event[0]](...event.splice(1))
		}

		// empty the array.
		meergo.length = 0

		globalThis.meergo = e
	})
}

main()
