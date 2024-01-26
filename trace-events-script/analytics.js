import Options from './options.js';
import Storage from './storage.js';
import Session from './session.js';
import Sender from './sender.js';
import { campaign, isPlainObject, typesOf, uuid } from './utils.js';

const version = '0.0.0';
const none = () => {};

class Analytics {
	#options;
	#storage;
	#session;
	#sender;
	#isReady = false;
	#onReady;
	#user = {
		id: (id) => {
			if (id === undefined) {
				return this.#storage.getUserID();
			}
			if (id === null) {
				this.#storage.setUserID(null);
				return null;
			}
			let data = {};
			this.#setUserId(data, id);
			if ('userId' in data) {
				return data.userId;
			}
			return null;
		},
		anonymousId: (id) => this.setAnonymousId(id),
		traits: (traits) => {
			if (traits !== void (0)) {
				this.#storage.setTraits(traits);
			}
			return this.#storage.getTraits();
		},
	};

	constructor(writeKey, endpoint, options) {
		this.#options = new Options(options);
		this.#storage = new Storage();
		this.#session = new Session(this.#storage, this.#options.sessions.autoTrack, this.#options.sessions.timeout);
		this.#sender = new Sender(writeKey, endpoint);
		this.#anonymousId(); // generates a new anonymous ID if one does not already exist
		const onReady = this.#onReady;
		if (onReady) {
			for (let i = 0; i < onReady.length; i++) {
				setTimeout(onReady[i]());
			}
			this.#onReady = void 0;
		}
		this.#isReady = true;
	}

	// alias sends an alias event.
	alias() {
		return this.#send('alias', this.#setAliasArguments, arguments);
	}

	endSession() {
		this.#session.end();
	}

	// getSessionId returns the current session ID, or null if there is no
	// session.
	getSessionId() {
		return this.#session.get();
	}

	// group sends a group event.
	group() {
		return this.#send('group', this.#setGroupArguments, arguments);
	}

	// identify sends an identify event.
	identify() {
		return this.#send('identify', this.#setIdentifyArguments, arguments);
	}

	// page sends a page event.
	page() {
		return this.#send('page', this.#setPageScreenArguments, arguments);
	}

	// ready calls callback after Analytics finishes initializing.
	ready(callback) {
		if (typeof callback !== 'function') {
			return;
		}
		if (this.#isReady) {
			setTimeout(callback);
			return;
		}
		this.#onReady = this.#onReady || [];
		this.#onReady.push(callback);
	}

	// reset resets the user and group identifiers, and traits removing them from the storage.
	// It also resets the Anonymous ID by generating a new one.
	reset() {
		this.#storage.setGroupID(null);
		this.#storage.setTraits(null);
		this.#storage.setUserID(null);
		this.#storage.setAnonymousID(uuid());
	}

	// screen sends a screen event.
	screen() {
		return this.#send('screen', this.#setPageScreenArguments, arguments);
	}

	// setAnonymousId sets the default Anonymous ID or, if id is undefined,
	// returns the default Anonymous ID.
	setAnonymousId(id) {
		if (id === undefined) {
			return this.#storage.getAnonymousID();
		}
		if (typeof id === 'number') {
			id = String(id);
		}
		if (typeof id === 'string' && id !== '') {
			this.#storage.setAnonymousID(id);
		} else {
			this.#storage.setAnonymousID(uuid());
		}
		return id;
	}

	// startSession starts a new session.
	startSession(id) {
		if (id) {
			if (typeof id !== 'number' || id % 1 !== 0) {
				throw new Error('sessionId must be a positive integer');
			}
		} else {
			id = null;
		}
		this.#session.start(id);
	}

	// track sends a track event.
	track() {
		return this.#send('track', this.#setTrackArguments, arguments);
	}

	// user returns the default user as a value with methods 'id', to get the
	// User ID, and 'anonymousId' to get the Anonymous ID.
	user() {
		return this.#user;
	}

	// anonymousId returns the anonymous ID. If the anonymous ID is null, it
	// creates and stores a new generated anonymous ID, then returns it.
	#anonymousId() {
		let id = this.#storage.getAnonymousID();
		if (id == null) {
			id = uuid();
			this.#storage.setAnonymousID(id);
		}
		return id;
	}

	// setAliasArguments sets the arguments for alias calls.
	// It writes the 'userId' and 'previousId' arguments into data and
	// returns the options.
	#setAliasArguments(data, a) {
		if (a.length === 0) {
			throw new Error('User is missing');
		}
		data.userId = this.#getAlias(a.shift());
		let options;
		switch (typesOf(a)) {
			case '':
				break;
			case 'string':
				data.previousId = this.#getAlias(a[0]);
				break;
			case 'object':
				options = a[0];
				break;
			case 'string,object':
				data.previousId = this.#getAlias(a[0]);
				options = a[1];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// setIdentifyArguments sets the arguments for identify calls.
	// It writes the 'userId' and 'traits' arguments into data and
	// returns the options.
	#setIdentifyArguments(data, a) {
		let options;
		switch (typesOf(a)) {
			case '':
				this.#setUserId(data);
				break;
			case 'string':
				this.#setUserId(data, a[0]);
				break;
			case 'object':
				this.#setUserId(data);
				this.#setTraits(data, a[0]);
				break;
			case 'string,object':
				this.#setUserId(data, a[0]);
				this.#setTraits(data, a[1]);
				break;
			case 'object,object':
				this.#setUserId(data);
				this.#setTraits(data, a[0]);
				options = a[1];
				break;
			case 'string,object,object':
				this.#setUserId(data, a[0]);
				this.#setTraits(data, a[1]);
				options = a[2];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// setGroupArguments sets the arguments for group calls.
	// It writes the 'groupId' and 'traits' arguments into data and
	// returns the options.
	#setGroupArguments(data, a) {
		if (a.length === 0) {
			throw new Error('Group is missing');
		}
		this.#setGroup(data, a.shift());
		let options;
		switch (typesOf(a)) {
			case '':
				break;
			case 'object':
				data.traits = a[0];
				break;
			case 'object,object':
				data.traits = a[0];
				options = a[1];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// setTrackArguments sets the arguments for track calls.
	// It writes the 'event' and 'properties' arguments into data and
	// returns the options.
	#setTrackArguments(data, a) {
		if (a.length === 0 || typeof a[0] != 'string') {
			throw new Error('Event name is missing');
		}
		data.event = a.shift();
		let options;
		switch (typesOf(a)) {
			case '':
				break;
			case 'object':
				data.properties = a[0];
				break;
			case 'object,object':
				data.properties = a[0];
				options = a[1];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// setPageScreenArguments sets the arguments for page and screen calls.
	// It writes the 'category', 'name', and 'properties' arguments into data
	// and returns the options.
	#setPageScreenArguments(data, a) {
		let options;
		switch (typesOf(a)) {
			case '':
				break;
			case 'string':
				data.name = a[0];
				break;
			case 'object':
				data.properties = a[0];
				break;
			case 'string,string':
				data.category = a[0];
				data.name = a[1];
				break;
			case 'string,object':
				data.name = a[0];
				data.properties = a[1];
				break;
			case 'object,object':
				data.properties = a[0];
				options = a[1];
				break;
			case 'string,string,object':
				data.category = a[0];
				data.name = a[1];
				data.properties = a[2];
				break;
			case 'string,object,object':
				data.name = a[0];
				data.properties = a[1];
				options = a[2];
				break;
			case 'string,string,object,object':
				data.category = a[0];
				data.name = a[1];
				data.properties = a[2];
				options = a[3];
				break;
			default:
				throw new Error('Invalid arguments');
		}
		return options;
	}

	// getAlias returns the userId or previousId arguments of the alias calls.
	#getAlias(id) {
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			return String(id);
		}
		id = this.#storage.getUserID();
		if (id == null) {
			return this.#storage.getAnonymousID();
		}
		return id;
	}

	// setGroup sets the groupId with id.
	#setGroup(data, id) {
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			data.groupId = String(id);
			this.#storage.setGroupID(data.groupId);
			return;
		}
		id = this.#storage.getGroupID();
		if (id != null) {
			data.groupId = id;
		}
	}

	// setTraits sets the traits merging the user traits with traits.
	#setTraits(data, traits) {
		data.traits = this.#storage.getTraits();
		if (traits !== undefined) {
			for (let k in traits) {
				const v = traits[k];
				if (v === undefined) {
					delete data.traits[k];
				} else {
					data.traits[k] = v;
				}
			}
		}
		this.#storage.setTraits(data.traits);
		data.traits = this.#storage.getTraits();
	}

	// setUserId sets the userId with id.
	#setUserId(data, id) {
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			data.userId = String(id);
			let userId = this.#storage.getUserID();
			if (userId !== data.userId) {
				this.#storage.setUserID(data.userId);
				this.#storage.setTraits({});
			}
			return;
		}
		id = this.#storage.getUserID();
		if (id != null) {
			data.userId = id;
		}
	}

	// send sends an event of the given type, setting the arguments args with
	// the setArgs function.
	#send(type, setArgs, args) {
		const self = this;
		function executor(resolve, reject) {
			let event;
			const data = { type };
			// ES5: "Array.from" is not available.
			args = Array.prototype.slice.call(args);
			let callback;
			if (args.length > 0 && typeof args[args.length - 1] === 'function') {
				callback = args.pop();
			}
			try {
				const options = setArgs.call(self, data, args);
				event = self.#sendEvent(data, options);
			} catch (error) {
				reject(error);
				return;
			}
			if (callback) {
				callback({ attempts: 1, event: event });
			}
			resolve({ attempts: 1, event: event });
		}
		if (typeof window.Promise === 'function') {
			return new Promise(executor);
		}
		executor(none, none);
	}

	// sendEvent sends an event with the given options.
	#sendEvent(event, options) {
		if (options && 'timestamp' in options) {
			if (options.timestamp !== void 0) {
				event.timestamp = options.timestamp;
			}
		} else {
			event.timestamp = new Date();
		}

		event.messageId = uuid();
		event.anonymousId = this.#anonymousId();

		const loc = window.location;

		const canonical = document.querySelector('link[rel="canonical"]');
		let pageURL = canonical ? canonical.getAttribute('href') : null;
		let path;
		if (pageURL == null || pageURL === '') {
			pageURL = loc.href;
			const p = pageURL.indexOf('#');
			if (p !== -1) {
				pageURL = pageURL.substring(0, p);
			}
			path = loc.pathname;
		} else {
			let u = pageURL;
			const p = u.indexOf('#');
			if (p !== -1) {
				u = u.substring(0, p);
			}
			path = u.substring(u.indexOf('/'));
		}

		const page = {
			path: path,
			referrer: document.referrer,
			search: loc.search,
			title: document.title,
			url: pageURL,
		};

		switch (event.type) {
			case 'track':
			case 'page':
			case 'screen':
				const p = isPlainObject(event.properties) ? event.properties : {};
				for (let k in page) {
					if (k in p) {
						const v = p[k];
						if (typeof v === 'string' && v !== '') {
							page[k] = p[k];
						}
					} else {
						p[k] = page[k];
					}
				}
				if (event.type === 'page') {
					if ('category' in event) {
						p.category = event.category;
					}
					if ('name' in event) {
						p.name = event.name;
					}
				}
				event.properties = p;
				this.#setUserId(event);
				break;
			case 'group':
				this.#setUserId(event);
		}

		const n = window.navigator;
		event.context = {
			library: {
				name: 'chichi.js',
				version: version,
			},
			locale: n.language || n.userLanguage,
			page: page,
			screen: {
				width: window.screen.width,
				height: window.screen.height,
				density: window.devicePixelRatio,
			},
			userAgent: n.userAgent,
		};

		const c = campaign();
		if (c) {
			event.context.campaign = c;
		}

		event.integrations = {};
		if (options && typeof options.integrations == 'object') {
			for (let n in options.integrations) {
				event.integrations[n] = options.integrations[n];
			}
		}

		for (let option in options) {
			if (option !== 'integrations' && options[option] !== void 0) {
				event.context[option] = options[option];
			}
		}

		const [sessionId, start] = this.#session.getFresh();
		if (sessionId != null) {
			event.context.sessionId = sessionId;
			if (start) {
				event.context.sessionStart = true;
			}
		}
		this.#sender.send(event);

		return event;
	}
}

export default Analytics;
