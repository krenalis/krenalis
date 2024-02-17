import { Analytics, EndpointURLError } from './analytics.js'
import { uuid } from './utils.js'

function main() {
	// Do nothing if the browser is not supported.
	if (!uuid) {
		return
	}

	const analytics = globalThis.chichianalytics

	let a
	try {
		a = new Analytics(analytics.key, analytics.url, analytics.options)
	} catch (error) {
		if (error instanceof EndpointURLError) {
			console.error(error.message)
			return
		}
		throw error
	}

	void a.ready(function () {
		const methods = [
			'alias',
			'anonymize',
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
			analytics[method] = a[method].bind(a)
		}

		for (let i = 0; i < analytics.length; i++) {
			const event = analytics[i]
			analytics[event[0]](...event.splice(1))
		}

		// empty the array.
		analytics.length = 0

		globalThis.chichianalytics = a
	})
}

main()
