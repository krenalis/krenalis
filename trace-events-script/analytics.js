import Storage from './storage.js';
import Sender from './sender.js';
import { utm, uuid } from './utils.js';

class Analytics {
	#storage;
	#sender;

	constructor(source, endpoint) {
		this.#storage = new Storage();
		this.#sender = new Sender(source, endpoint);
	}

	// page sends a page event.
	page() {
		this.#sendEvent({ event: 'page' });
	}

	// identify sends an identify event.
	identify(id, traits) {
		this.#storage.setUserID(id);

		if (traits) {
			this.#storage.setUserTraits(traits);
		}

		this.#sendEvent({ event: 'identify' });
	}

	// track sends a track event.
	track(event, properties) {
		if (typeof event !== 'string' || event === '') {
			throw new Error('Event name is missing');
		}
		this.#sendEvent({ event: event, properties: properties });
	}

	// sendEvent sends an event.
	#sendEvent(data) {
		const event = data == null ? {} : data;

		let anonymousId = this.#storage.getAnonymousId();
		if (anonymousId == null) {
			anonymousId = uuid();
			this.#storage.setAnonymousId(anonymousId);
		}
		event.anonymousId = anonymousId;

		event.referrer = document.referrer;

		const n = navigator;
		if (n.connection) {
			event.connection = n.connection.type == null ? '' : n.connection.type;
		}
		if (n.language) {
			event.language = n.language;
		}
		if (n.userAgentData) {
			event.isMobile = n.userAgentData.mobile;
		}

		event.title = document.title;
		event.url = window.location.href;
		event.screen = {
			density: window.devicePixelRatio,
			width: window.screen.width,
			height: window.screen.height,
		};

		const u = utm();
		if (u !== undefined) {
			event.utm = u;
		}

		if ('properties' in event) {
			event.properties = asProperties(event.properties);
		}

		this.#sender.send(event);
	}
}

// asProperties returns properties if asProperties is representable as a JSON
// object, otherwise returns {}.
function asProperties(properties) {
	return isPlainObject(properties) ? properties : {};
}

// isPlainObject reports whether obj is a plain object.
function isPlainObject(obj) {
	return typeof obj === 'object' && !Array.isArray(obj) && obj != null;
}

export default Analytics;
