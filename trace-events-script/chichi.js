(() => {
	
	// set DEBUG to true if you are running Chichi's server on localhost.
	const DEBUG = false;

	if (navigator == null) {
		console.error(`[ChichiError] cannot run Chichi: browser doesn't support the Navigator interface`);
		return;
	}

	let script = document.querySelector('script[data-isChichi]');
	let endpoint = DEBUG ? 'https://localhost:9090/log-event' : script.src; // TODO(@Andrea): replace the last segment of script.src with 'log-event'
	let property = script.dataset.property;

	// immediately send the pageView event.
	sendEvent({ event: 'pageview' });

	// when the DOM is fully parsed, attach the event listeners to all the
	// clickable elements.
	addEventListener('DOMContentLoaded', () => {
		let clickableElements = document.querySelectorAll('a, button, input[type="submit"]');
		for (let el of clickableElements) { el.addEventListener('click', sendClickEvent, false) }
	});

	function sendEvent(data) {
		let d = data == null ? {} : data;

		d.property = property;

		let device;
		if (isStorageAvailable('localStorage')) {
			device = localStorage.getItem('chichi-device');
			if (device == null) {
				device = makeid();
				localStorage.setItem('chichi-device', device);
			}
		}

		d.referrer = document.referrer;

		let n = navigator;
		if (n.connection) d.connection = n.connection.type == null ? '' : n.connection.type;
		if (n.language) d.language = n.language;
		if (n.userAgentData) d.isMobile = n.userAgentData.mobile;

		d.title = document.title;

		d.url = window.location.href;

		console.log(d);

		// serialize the collected data in JSON.
		let json;
		try {
			json = JSON.stringify(d);
		} catch (err) {
			console.error(`[ChichiError] cannot parse the collected data: ${err.message}`)
		}

		// send the JSON.
		try {
			n.sendBeacon(endpoint, json);
		} catch (err) {
			console.error(`[ChichiError] cannot send '${d.event}' event: ${err.message}`)
		}
	}

	function sendClickEvent(e) {
		let text, target, event, initiator;
		switch (e.currentTarget.tagName) {
			case "A":
				[text, target, event] = getAnchorData(e.currentTarget);
				initiator = 'anchor';
				break;
			case "INPUT":
				[text, target, event] = GetInputSubmitData(e.currentTarget);
				initiator = 'input of type submit';
				break;
			case "BUTTON":
				[text, target, event] = getButtonData(e.currentTarget);
				initiator = 'button';
				break;
			default:
				console.error(`[ChichiError] unexpected clickable element with tag '${e.currentTarget.tagName}'`);
				return;
		}
		sendEvent({ event: event, target: target, text: text, initiator: initiator });
	}

	function getAnchorData(a) {
		let text = a.textContent;
		let target = a.href;
		let event = 'click';
		return [text, target, event];
	}

	function GetInputSubmitData(i) {
		let text = i.value;
		let target, event;
		let form, formID;
		if ((formID = i.getAttribute('form')) != null) {
			form = document.querySelector(`form#${formID}`);
		} else {
			form = i.closest('form');
		}
		if (form == null) return [text, '', 'click'];
		target = form.action;
		formaction = i.getAttribute('formaction');
		if (formaction != null && formaction != target) target = formaction;
		event = 'form submission';
		return [text, target, event];
	}

	function getButtonData(b) {
		let text = b.textContent;
		let target, event;
		let isSubmit = b.type == 'submit';
		if (!isSubmit) return [text, '', 'click'];
		let form, formID;
		if ((formID = b.getAttribute('form')) != null) {
			form = document.querySelector(`form#${formID}`);
		} else {
			form = b.closest('form');
		}
		if (form == null) return [text, '', 'click'];
		target = form.action;
		formaction = b.getAttribute('formaction');
		if (formaction != null && formaction != target) target = formaction;
		event = 'form submission';
		return [text, target, event];
	}

	function isStorageAvailable(type) {
		let storage;
		try {
			storage = window[type];
			const x = '__storage_test__';
			storage.setItem(x, x);
			storage.removeItem(x);
			return true;
		}
		catch (e) {
			return e instanceof DOMException && (
				// everything except Firefox
				e.code === 22 ||
				// Firefox
				e.code === 1014 ||
				// test name field too, because code might not be present
				// everything except Firefox
				e.name === 'QuotaExceededError' ||
				// Firefox
				e.name === 'NS_ERROR_DOM_QUOTA_REACHED') &&
				// acknowledge QuotaExceededError only if there's something already stored
				(storage && storage.length !== 0);
		}
	}

	function makeid() {
		let array = new Uint8Array(20);
		self.crypto.getRandomValues(array);
		return btoa(String.fromCharCode.apply(null, array));
	}

})();
