import Storage from './storage.js';
import Sender from './sender.js';
import { campaign, uuid, typesOf } from './utils.js';

const version = '0.0.0';

class Analytics {
	#storage;
	#sender;
	#isReady = false;
	#onReady;
	#user = {
		id: (id) => this.#setUserId(id),
		anonymousId: (id) => this.setAnonymousId(id),
	};

	constructor(source, endpoint) {
		this.#storage = new Storage();
		this.#sender = new Sender(source, endpoint);
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

	// getAnonymousId returns the default Anonymous ID.
	getAnonymousId() {
		return this.#storage.getAnonymousID();
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

	// reset resets the user and group identifiers removing them from the storage.
	reset() {
		this.#storage.reset();
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
		if (!id) {
			this.#storage.setAnonymousID(uuid());
			return null;
		}
		this.#storage.setAnonymousID(id);
		return id;
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
				this.#setUser(data);
				break;
			case 'string':
				this.#setUser(data, a[0]);
				break;
			case 'object':
				this.#setUser(data);
				data.traits = a[0];
				break;
			case 'string,object':
				this.#setUser(data, a[0]);
				data.traits = a[1];
				break;
			case 'object,object':
				this.#setUser(data);
				data.traits = a[0];
				options = a[1];
				break;
			case 'string,object,object':
				this.#setUser(data, a[0]);
				data.traits = a[1];
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

	// setUserId sets the default User ID or, if id is undefined,
	// returns the default User ID.
	#setUserId(id) {
		if (id === undefined) {
			return this.#storage.getUserID();
		}
		if (!id) {
			id = null;
		}
		this.#storage.setUserID(id);
		return id;
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

	// setUser sets the userId with id.
	#setUser(data, id) {
		if ((typeof id === 'string' && id !== '') || typeof id === 'number') {
			data.userId = String(id);
			this.#storage.setUserID(data.userId);
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
		return new Promise(function (resolve, reject) {
			let event;
			const data = { type };
			// Legacy: ie10 and ie11 do not support Array.from.
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
			}
			if (callback) {
				callback({ attempts: 1, event: event });
			}
			resolve({ attempts: 1, event: event });
		});
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

		let anonymousID = this.#storage.getAnonymousID();
		if (anonymousID == null) {
			anonymousID = uuid();
			this.#storage.setAnonymousID(anonymousID);
		}
		event.anonymousId = anonymousID;

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
				this.#setUser(event);
				break;
			case 'group':
				this.#setUser(event);
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
				density: window.devicePixelRatio,
				width: window.screen.width,
				height: window.screen.height,
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

		this.#sender.send(event);

		return event;
	}
}

// isPlainObject reports whether obj is a plain object.
function isPlainObject(obj) {
	return typeof obj === 'object' && !Array.isArray(obj) && obj != null;
}

export default Analytics;
