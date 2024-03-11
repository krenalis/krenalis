import { assertEquals } from 'std/assert/mod.ts'
import { FakeTime } from 'std/testing/time.ts'
import Storage from './storage.js'
import Session from './session.js'
import Options from './options.js'

const DEBUG = false

const writeKey = 'rq6JJg5ENWK28NHfxSwJZmzeIvDC8GQO'

Deno.test('Session', async (t) => {
	const fiveMinutes = 5 * 60 * 1000
	const tenMinutes = 2 * fiveMinutes

	let session

	function expectGet(id) {
		assertEquals(session.get(), id)
	}

	function expectGetFresh(id, start) {
		const [actualID, actualStart] = session.getFresh() // autoTrack is false.
		assertEquals(actualStart, start)
		assertEquals(actualID, id)
	}

	await t.step('session without auto tracking', () => {
		localStorage.clear()
		const time = new FakeTime()

		const options = new Options()
		options.stores = ['localStorage']
		const storage = new Storage(writeKey, options)
		session = new Session(storage, false, tenMinutes)
		session.debug(DEBUG)

		// Check that the session is not started.
		expectGet(null)
		expectGetFresh(null, false)

		session.start()
		const startedAt = new Date().getTime()
		expectGet(startedAt)
		expectGetFresh(startedAt, true)
		expectGetFresh(startedAt, false)

		session.end()
		expectGet(null)

		const newSessionID = 52089473
		session.start(newSessionID)
		expectGet(newSessionID)
		expectGetFresh(newSessionID, true)

		time.tick(tenMinutes * 2)
		expectGetFresh(newSessionID, false)

		session.end()
		expectGet(null)

		expectGetFresh(null, false)

		time.restore()
	})

	await t.step('session with auto tracking', () => {
		localStorage.clear()
		const time = new FakeTime()

		const startedAt = new Date().getTime()

		const options = new Options()
		options.stores = ['localStorage']
		const storage = new Storage(writeKey, options)
		session = new Session(storage, true, tenMinutes)
		session.debug(DEBUG)

		assertEquals(session.get(), startedAt)
		expectGetFresh(startedAt, true) // start is true because will be the first call to getFresh.
		expectGetFresh(startedAt, false)

		time.tick(fiveMinutes) // advance 5 minutes in time
		expectGet(startedAt)

		time.tick(fiveMinutes) // advance another 5 minutes in time
		expectGet(startedAt)

		time.tick(1) // advance 1ms in time
		expectGet(null)
		expectGetFresh(startedAt + 2 * fiveMinutes + 1, true)

		session.end()
		expectGet(null)

		time.tick(fiveMinutes) // advance another 5 minutes in time
		expectGetFresh(startedAt + 3 * fiveMinutes + 1, true)
		expectGet(startedAt + 3 * fiveMinutes + 1)

		time.restore()
	})
})
