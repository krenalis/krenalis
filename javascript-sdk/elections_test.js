import { assert, assertEquals, AssertionError } from 'std/assert/mod.ts'
import { FakeTime } from 'std/testing/time.ts'
import { uuid } from './utils.js'
import Elections from './elections.js'

const DEBUG = false

Deno.test('Elections: keep', async () => {
	const aID = uuid()
	const bID = uuid()

	const tests = [
		{
			steps: [
				'a:r1',
				'a:w1',
				'a:r2',
				'a:w2',
				'a:r1', // 'a' become the leader
				'b:r1', // 'b' gives up ('a' owns location 1)
				1000,
				'a:r1', // 'a' tries to renew its leadership
				'a:w1', // 'a' renews its leadership
				400,
				'b:r1', // 'b' gives up ('a' owns location 1)
			],
			elections: ['a:true', 'b:false', 'a:true', 'b:false'],
		},
		{
			steps: [
				'a:r1',
				'b:r1',
				'a:w1',
				'a:r2',
				'a:w2',
				'a:r1', // 'a' become the leader
				'b:w1',
				'b:r2', // 'b' gives up ('a' owns location 2), but it owns location 1
				1000,
				'a:r1', // 'a' gives up ('b' owns location 1)
				400,
				'b:r1',
				'b:w1',
				'b:r2',
				'b:w2',
				'b:r1', // 'b' become the leader
				'a:r1', // 'a' gives up ('b' owns location 1)
			],
			elections: ['a:true', 'b:false', 'a:false', 'b:true', 'a:false'],
		},
		{
			steps: [
				'b:r1',
				'a:r1',
				'a:w1',
				'a:r2',
				'b:w1',
				'b:r2',
				'b:w2',
				'a:w2',
				'a:r1', // 'a' gives up ('b' owns location 1)
				'b:r1', // 'b' become the leader
			],
			elections: ['a:false', 'b:true'],
		},
		{
			steps: [
				'a:r1',
				'a:w1',
				'a:r2',
				'a:w2',
				'a:r1', // 'a' become the leader
				'b:r1', // 'b' gives up ('a' owns location 1)
				1400,
				'b:r1',
				'b:w1',
				'b:r2',
				'b:w2',
				'b:r1', // 'b' become the leader
				'a:r1', // 'a' gives up ('b' owns location 1)
			],
			elections: ['a:true', 'b:false', 'b:true', 'a:false'],
		},
	]

	for (let i = 0; i < tests.length; i++) {
		const time = new FakeTime(0)
		try {
			const test = tests[i]
			const state = new ElectionState(test.steps, { a: aID, b: bID }, time)
			const a = new Elections(aID, state, (isLeader) => {
				if (DEBUG) {
					console.log(isLeader ? `--> elected as leader: a` : `--> removed form leader: a`)
				}
				assert(test.elections.length > 0, `candidate 'a' notified an unexpected election`)
				const election = test.elections.shift()
				assertEquals(
					`a:${isLeader}`,
					election,
					`candidate 'a' notified a wrong election, expected '${election}', got 'a:${isLeader}'`,
				)
			})
			const b = new Elections(bID, state, (isLeader) => {
				if (DEBUG) {
					console.log(isLeader ? `--> elected as leader: b` : `--> removed form leader: b`)
				}
				assert(test.elections.length > 0, `candidate 'b' notified an unexpected election`)
				const election = test.elections.shift()
				assertEquals(
					`b:${isLeader}`,
					election,
					`candidate 'b' notified a wrong election, expected '${election}', got 'b:${isLeader}'`,
				)
			})
			time.tick(0)
			await state.closed()
			a.close()
			b.close()
		} finally {
			time.restore()
		}
	}
})

Deno.test('Elections: resign', async () => {
	const time = new FakeTime(0)
	const aID = uuid()
	const state = new ElectionState(
		[
			'a:r1',
			'a:w1',
			'a:r2',
			'a:w2',
			'a:r1',
			'a:r1',
			'a:w1',
		],
		{ a: aID },
		time,
	)
	const elections = ['a:true', 'a:false']
	const a = new Elections(aID, state, (isLeader) => {
		if (DEBUG) {
			console.log(isLeader ? `--> elected as leader: a` : `--> removed form leader: a`)
		}
		assert(elections.length > 0, `candidate 'a' notified an unexpected election`)
		const election = elections.shift()
		assertEquals(
			`a:${isLeader}`,
			election,
			`candidate 'a' notified a wrong election, expected '${election}', got 'a:${isLeader}'`,
		)
		if (isLeader) {
			setTimeout(a.resign.bind(a))
		}
	})
	time.tick(0)
	await state.closed()
	assertEquals(elections.length, 0)
	a.close()
})

class ElectionState {
	#steps
	#candidates
	#time
	#state = []
	#waiting = {}
	#close
	#closed

	constructor(steps, candidates, time) {
		for (let i = 0; i < steps.length; i++) {
			const step = steps[i]
			if (typeof step === 'string') {
				steps[i] = new Step(step)
				assert(steps[i].candidate in candidates)
			} else {
				assert(typeof step, 'number')
			}
		}
		this.#steps = steps
		this.#candidates = candidates
		this.#time = time
		this.#closed = new Promise((resolve) => {
			this.#close = resolve
		})
	}

	// closed returns a promise that resolves when the instance is closed,
	// meaning all steps have been executed.
	closed() {
		return this.#closed
	}

	read(id, location, cb) {
		const candidate = this.#candidateByID(id)
		if (DEBUG) {
			console.log(`request from ${candidate}: read(${location})`)
		}
		assert(location === 1 || location === 2, `unknown location ${location}`)
		assertEquals(typeof cb, 'function', `callback has type ${typeof cb}, not 'function'`)
		this.#do(new Step(`${candidate}:r${location}`), null, cb)
	}

	write(id, location, value, cb) {
		const candidate = this.#candidateByID(id)
		if (DEBUG) {
			console.log(`request from ${candidate}: write(${location}, '${value}')`)
		}
		assert(location === 1 || location === 2, `unknown location ${location}`)
		assertEquals(typeof value, 'string', `value has type ${typeof value}, not 'string'`)
		assertEquals(typeof cb, 'function', `callback has type ${typeof cb}, not 'function'`)
		this.#do(new Step(`${candidate}:w${location}`), value, cb)
	}

	#candidateByID(id) {
		assert(typeof id === 'string')
		for (const candidate in this.#candidates) {
			if (this.#candidates[candidate] === id) {
				return candidate
			}
		}
		throw new AssertionError(`unknown candidate '${id}'`)
	}

	#do(step, value, cb) {
		assert(
			!(step.candidate in this.#waiting),
			`received step ${step.toString()}, however, a step of the same candidate is still pending execution`,
		)
		let wait
		for (let i = 0; i < this.#steps.length; i++) {
			const s = this.#steps[i]
			assert(
				typeof s !== 'number',
				`received step ${step.toString()}, however, even if it were among the expected steps, it would still be preceded by a time increment`,
			)
			if (step.candidate === s.candidate) {
				assertEquals(
					step,
					s,
					`received step ${step.toString()}, however, step ${step.toString()} of the same candidate was expected first`,
				)
				wait = i > 0
				break
			}
		}
		assert(wait != null, `unexpected step ${step}`)
		const serveFirst = () => {
			const s = this.#steps.shift()
			if (DEBUG) {
				console.log(`serving ${s.toString()}`)
			}
			if (step.operation === 'r') {
				setTimeout(cb, 0, this.#state[step.location - 1])
				return
			}
			this.#state[step.location - 1] = value
			setTimeout(cb)
		}
		if (wait) {
			this.#waiting[step.candidate] = serveFirst
			return
		}
		serveFirst()
		while (this.#steps.length > 0) {
			const s = this.#steps[0]
			if (typeof s === 'number') {
				const ms = this.#steps.shift()
				if (DEBUG) {
					console.log(`advance time by ${ms}ms`)
				}
				this.#time.tick(ms)
				continue
			}
			const serveFirst = this.#waiting[s.candidate]
			if (serveFirst == null) {
				break
			}
			delete (this.#waiting[s.candidate])
			serveFirst()
		}
		if (this.#steps.length === 0) {
			setTimeout(() => this.#close())
		}
	}
}

class Step {
	candidate
	operation
	location
	constructor(s) {
		const m = s.match(/^([a-z]):([rw])([12])$/)
		if (m == null) {
			throw new Error(`malformed step: ${s}`)
		}
		this.candidate = m[1]
		this.operation = m[2]
		this.location = Number(m[3])
	}
	toString() {
		return `${this.candidate}:${this.operation}${this.location}`
	}
}
