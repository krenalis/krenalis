// Steps executed by the 'analytics_test.js' tests.

const timestamp = '2024-01-01T00:00:01.000Z'
const messageId = '9587b6d1-ae92-4d3c-a8d9-87c3e9ce7ae3'
const anonymousId = '1b82c7e4-00b7-45d1-bbe2-6375fa9f8fa7'
const path = '/path'
const referrer = ''
const search = '?query=123'
const title = 'Hello from Meergo'
const url = 'https://example.com:8080/path?query=123'
const properties = { path, referrer, search, title, url }
const page = { path, referrer, search, title, url }
const library = { name: 'meergo.js', version: '0.0.0' }
const screen = { width: 2560, height: 1440, density: 1.25 }
const timezone = 'America/New_York'
const userAgent =
	'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36'
const locale = 'en-US'
const sessionId = 1704070861000
const context = { library, locale, page, screen, sessionId, timezone, userAgent }
const integrations = {}
const traits = {}
const userId = null

const steps = [
	// Page.
	{
		name: `page()`,
		call: (meergo) => {
			meergo.page()
		},
		event: { type: 'page', timestamp, messageId, anonymousId, properties, context, integrations, userId },
	},
	{
		name: `page(name)`,
		call: (meergo) => {
			meergo.page('Meergo Home')
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			name: 'Meergo Home',
			anonymousId,
			properties: { name: 'Meergo Home', path, referrer, search, title, url },
			context,
			integrations,
			userId,
		},
	},
	{
		name: `page(properties)`,
		call: (meergo) => {
			meergo.page({ title: 'alternative title', foo: 'boo' })
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			anonymousId,
			properties: { path, referrer, search, title: 'alternative title', url, foo: 'boo' },
			context: {
				library,
				locale,
				page: { path, referrer, search, title: 'alternative title', url },
				screen,
				sessionId,
				timezone,
				userAgent,
			},
			integrations,
			userId,
		},
	},
	{
		name: `page(category, name)`,
		call: (meergo) => {
			meergo.page('Products', 'Shirt')
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			name: 'Shirt',
			category: 'Products',
			anonymousId,
			properties: { name: 'Shirt', category: 'Products', path, referrer, search, title, url },
			context,
			integrations,
			userId,
		},
	},
	{
		name: `page(name, properties)`,
		call: (meergo) => {
			meergo.page('Sign Up', { resellers: true })
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			name: 'Sign Up',
			anonymousId,
			properties: { name: 'Sign Up', path, referrer, search, title, url, resellers: true },
			context,
			integrations,
			userId,
		},
	},
	{
		name: `page(properties, options)`,
		call: (meergo) => {
			meergo.page({ title: 'alternative title', foo: 'boo' }, { count: 150 })
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			anonymousId,
			properties: { path, referrer, search, title: 'alternative title', url, foo: 'boo' },
			context: {
				count: 150,
				library,
				locale,
				page: { path, referrer, search, title: 'alternative title', url },
				screen,
				sessionId,
				timezone,
				userAgent,
			},
			integrations,
			userId,
		},
	},
	{
		name: `page(category, name, properties)`,
		call: (meergo) => {
			meergo.page('users', 'Sign Up', { resellers: true })
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			name: 'Sign Up',
			category: 'users',
			anonymousId,
			properties: { name: 'Sign Up', category: 'users', path, referrer, search, title, url, resellers: true },
			context,
			integrations,
			userId,
		},
	},
	{
		name: `page(name, properties, options)`,
		call: (meergo) => {
			meergo.page('', { data: { a: 'b' } }, { locale: 'it-IT' })
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			name: '',
			anonymousId,
			properties: { path, referrer, search, title, url, data: { a: 'b' } },
			context: { library, locale: 'it-IT', page, screen, sessionId, timezone, userAgent },
			integrations,
			userId,
		},
	},
	{
		name: `page(category, name, properties, options)`,
		call: (meergo) => {
			meergo.page('videos', 'cats', { data: { a: 'b' } }, { locale: 'it-IT' })
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			name: 'cats',
			category: 'videos',
			anonymousId,
			properties: { path, referrer, search, title, url, name: 'cats', category: 'videos', data: { a: 'b' } },
			context: { library, locale: 'it-IT', page, screen, sessionId, timezone, userAgent },
			integrations,
			userId,
		},
	},
	// Screen.
	{
		name: `screen()`,
		call: (meergo) => {
			meergo.screen()
		},
		event: { type: 'screen', timestamp, messageId, anonymousId, properties: {}, context, integrations, userId },
	},
	{
		name: `screen(name)`,
		call: (meergo) => {
			meergo.screen('Meergo Main')
		},
		event: {
			type: 'screen',
			timestamp,
			messageId,
			name: 'Meergo Main',
			anonymousId,
			properties: {},
			context,
			integrations,
			userId,
		},
	},
	{
		name: `screen(properties)`,
		call: (meergo) => {
			meergo.screen({ score: 517836 })
		},
		event: {
			type: 'screen',
			timestamp,
			messageId,
			anonymousId,
			properties: { score: 517836 },
			context,
			integrations,
			userId,
		},
	},
	{
		name: `screen(category, name)`,
		call: (meergo) => {
			meergo.screen('Products', 'Shirt')
		},
		event: {
			type: 'screen',
			timestamp,
			messageId,
			name: 'Shirt',
			category: 'Products',
			anonymousId,
			properties: {},
			context,
			integrations,
			userId,
		},
	},
	{
		name: `screen(name, properties)`,
		call: (meergo) => {
			meergo.screen('Sign Up', { resellers: true })
		},
		event: {
			type: 'screen',
			timestamp,
			messageId,
			name: 'Sign Up',
			anonymousId,
			properties: { resellers: true },
			context,
			integrations,
			userId,
		},
	},
	{
		name: `screen(properties, options)`,
		call: (meergo) => {
			meergo.screen({ step: 6 }, { count: 150 })
		},
		event: {
			type: 'screen',
			timestamp,
			messageId,
			anonymousId,
			properties: { step: 6 },
			context: {
				count: 150,
				library,
				locale,
				page: { path, referrer, search, title, url },
				screen,
				sessionId,
				timezone,
				userAgent,
			},
			integrations,
			userId,
		},
	},
	{
		name: `screen(category, name, properties)`,
		call: (meergo) => {
			meergo.screen('users', 'Sign Up', { resellers: true })
		},
		event: {
			type: 'screen',
			timestamp,
			messageId,
			name: 'Sign Up',
			category: 'users',
			anonymousId,
			properties: { resellers: true },
			context,
			integrations,
			userId,
		},
	},
	{
		name: `screen(name, properties, options)`,
		call: (meergo) => {
			meergo.screen('', { data: { a: 'b' } }, { locale: 'it-IT' })
		},
		event: {
			type: 'screen',
			timestamp,
			messageId,
			name: '',
			anonymousId,
			properties: { data: { a: 'b' } },
			context: { library, locale: 'it-IT', page, screen, sessionId, timezone, userAgent },
			integrations,
			userId,
		},
	},
	{
		name: `screen(category, name, properties, options)`,
		call: (meergo) => {
			meergo.screen('videos', 'cats', { data: { a: 'b' } }, { locale: 'it-IT' })
		},
		event: {
			type: 'screen',
			timestamp,
			messageId,
			name: 'cats',
			category: 'videos',
			anonymousId,
			properties: { data: { a: 'b' } },
			context: { library, locale: 'it-IT', page, screen, sessionId, timezone, userAgent },
			integrations,
			userId,
		},
	},
	// Track.
	{
		name: `track()`,
		call: (meergo) => {
			return meergo.track()
		},
		error: new Error('Event name is missing'),
	},
	{
		name: `track(event)`,
		call: (meergo) => {
			meergo.track('Click')
		},
		event: {
			type: 'track',
			event: 'Click',
			timestamp,
			messageId,
			anonymousId,
			properties: {},
			context,
			integrations,
			userId,
		},
	},
	{
		name: `track(event, properties)`,
		call: (meergo) => {
			meergo.track('Product Viewed', { productId: 819382 })
		},
		event: {
			type: 'track',
			event: 'Product Viewed',
			timestamp,
			messageId,
			anonymousId,
			properties: { productId: 819382 },
			context,
			integrations,
			userId,
		},
	},
	{
		name: `track(event, properties, options)`,
		call: (meergo) => {
			meergo.track('Product Viewed', { productId: 819382 }, { locale: 'it-IT' })
		},
		event: {
			type: 'track',
			event: 'Product Viewed',
			timestamp,
			messageId,
			anonymousId,
			properties: { productId: 819382 },
			context: { library, locale: 'it-IT', page, screen, sessionId, timezone, userAgent },
			integrations,
			userId,
		},
	},
	// Identify.
	{
		name: `identify()`,
		call: (meergo) => {
			meergo.identify()
		},
		event: {
			type: 'identify',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits,
			userId,
		},
	},
	{
		name: `identify(userId)`,
		call: (meergo) => {
			meergo.identify('920577314')
		},
		event: {
			type: 'identify',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits,
			userId: '920577314',
		},
	},
	{
		name: `identify(userId) // with anonymous traits`,
		call: (meergo) => {
			meergo.user().traits({ first_name: 'Susan', last_name: 'Davis' })
			meergo.identify('920577314')
		},
		event: {
			type: 'identify',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: { first_name: 'Susan', last_name: 'Davis' },
			userId: '920577314',
		},
	},
	{
		name: `identify(null)`,
		call: (meergo) => {
			meergo.user().id('920577314')
			meergo.user().traits({ first_name: 'Susan' })
			meergo.identify(null)
		},
		event: {
			type: 'identify',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: { first_name: 'Susan' },
			userId: '920577314',
		},
	},
	{
		name: `identify(traits)`,
		call: (meergo) => {
			meergo.identify({ first_name: 'Susan', last_name: 'Davis' })
		},
		event: {
			type: 'identify',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: { first_name: 'Susan', last_name: 'Davis' },
			userId,
		},
	},
	{
		name: `identify(userId, traits)`,
		call: (meergo) => {
			meergo.identify('920577314', { first_name: 'Susan', last_name: 'Davis' })
		},
		event: {
			type: 'identify',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: { first_name: 'Susan', last_name: 'Davis' },
			userId: '920577314',
		},
	},
	{
		name: `identify(null, traits)`,
		call: (meergo) => {
			meergo.user().id('920577314')
			meergo.user().traits({ first_name: 'Susan' })
			meergo.identify(null, { last_name: 'Davis' })
		},
		event: {
			type: 'identify',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: { first_name: 'Susan', last_name: 'Davis' },
			userId: '920577314',
		},
	},
	{
		name: `identify(traits, options)`,
		call: (meergo) => {
			meergo.identify({ first_name: 'Susan', last_name: 'Davis' }, { locale: 'it-IT', key: 'value' })
		},
		event: {
			type: 'identify',
			timestamp,
			messageId,
			anonymousId,
			context: { library, locale: 'it-IT', page, screen, sessionId, userAgent, timezone, key: 'value' },
			integrations,
			traits: { first_name: 'Susan', last_name: 'Davis' },
			userId,
		},
	},
	{
		name: `identify(userId, traits, options)`,
		call: (meergo) => {
			meergo.identify(603614922, { age: 36 }, { locale: 'it-IT', key: 'value' })
		},
		event: {
			type: 'identify',
			timestamp,
			messageId,
			anonymousId,
			context: { library, locale: 'it-IT', page, screen, sessionId, userAgent, timezone, key: 'value' },
			integrations,
			traits: { age: 36 },
			userId: '603614922',
		},
	},
	{
		name: `identify(userId, userId)`,
		call: (meergo) => {
			return meergo.identify(603614922, 603614922)
		},
		error: new Error('Invalid arguments'),
	},
	// Group.
	{
		name: `group(groupId)`,
		call: (meergo) => {
			meergo.group('3617408')
		},
		event: {
			type: 'group',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: {},
			userId: null,
			groupId: '3617408',
		},
	},
	{
		name: `group(undefined)`,
		call: (meergo) => {
			meergo.group(undefined)
		},
		event: {
			type: 'group',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits,
			groupId: null,
			userId,
		},
	},
	{
		name: `group(null)`,
		call: (meergo) => {
			meergo.group().id('acme')
			meergo.group().traits({ name: 'Acme' })
			meergo.group(null)
		},
		event: {
			type: 'group',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: { name: 'Acme' },
			groupId: 'acme',
			userId,
		},
	},
	{
		name: `group(traits)`,
		call: (meergo) => {
			meergo.group({ name: 'Acme Inc.' })
		},
		event: {
			type: 'group',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: { name: 'Acme Inc.' },
			userId,
		},
	},
	{
		name: `group(groupId, traits)`,
		call: (meergo) => {
			meergo.group(3617408, { name: 'Acme Inc.' })
		},
		event: {
			type: 'group',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: { name: 'Acme Inc.' },
			groupId: '3617408',
			userId,
		},
	},
	{
		name: `group(null, traits)`,
		call: (meergo) => {
			meergo.group().id('acme')
			meergo.group().traits({ name: 'Acme' })
			meergo.group(null, { employees: 85 })
		},
		event: {
			type: 'group',
			timestamp,
			messageId,
			anonymousId,
			context,
			integrations,
			traits: { name: 'Acme', employees: 85 },
			groupId: 'acme',
			userId,
		},
	},
	{
		name: `group(traits, options)`,
		call: (meergo) => {
			meergo.group({ name: 'Acme Inc.' }, { k: true })
		},
		event: {
			type: 'group',
			timestamp,
			messageId,
			anonymousId,
			context: { library, locale, page, screen, sessionId, timezone, userAgent, k: true },
			integrations,
			traits: { name: 'Acme Inc.' },
			userId,
		},
	},
	{
		name: `group(groupId, traits, options)`,
		call: (meergo) => {
			meergo.group('3617408', { name: 'Acme Inc.' }, { k: true })
		},
		event: {
			type: 'group',
			timestamp,
			messageId,
			anonymousId,
			context: { library, locale, page, screen, sessionId, timezone, userAgent, k: true },
			integrations,
			traits: { name: 'Acme Inc.' },
			groupId: '3617408',
			userId,
		},
	},
	// Sessions.
	{
		name: `no session`,
		options: { sessions: { autoTrack: false } },
		call: (meergo) => {
			meergo.page()
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			anonymousId,
			properties,
			context: { library, locale, page, screen, timezone, userAgent },
			integrations,
			userId,
		},
	},
	{
		name: `session started`,
		options: { sessions: { autoTrack: false } },
		call: (meergo) => {
			meergo.startSession(1508273)
			meergo.page()
		},
		event: {
			type: 'page',
			timestamp,
			messageId,
			anonymousId,
			properties,
			context: { library, locale, page, screen, sessionId: 1508273, sessionStart: true, timezone, userAgent },
			integrations,
			userId,
		},
	},
]

export { steps }
