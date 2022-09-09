let endpoint = 'https://localhost:9090/log-event';

let property = '1234567890'

function sendBeacon(data) {
	let d = data == null ? {} : data;

	// populate the data object with relevant infos.
	let device = localStorage.getItem('chichi-device');
	if (device == null) {
		device = makeid();
		localStorage.setItem('chichi-device', device);
	}
	d.device = device;
	d.property = property;
	navigator.geolocation.getCurrentPosition(
		(position) => {
			d.geolocation = {
				latitude: position.coords.latitude,
				longitude: position.coords.longitude,
			};
		},
		(err) => {
			console.log(`cannot get the geolocation: ${err.message}`);
		}
	);
	d.referrer = document.referrer;
	if (navigator.connection) {
		d.connection = navigator.connection.type == null ? '' : navigator.connection.type;
	}
	d.language = navigator.language;
	if (navigator.userAgentData) {
		d.os = navigator.userAgentData.platform;
		d.isMobile = navigator.userAgentData.mobile;
	}
	d.title = document.title;
	d.url = window.location.href;

	console.log(d);

	// serialize the data.
	let json;
	try {
		json = JSON.stringify(d);
	} catch (err) {
		console.error(`cannot parse Chichi's data`);
	}

	// send the JSON.
	navigator.sendBeacon(endpoint, json);
}

window.addEventListener(
	'load',
	(e) => {
		sendBeacon({
			event: 'pageview',
		});
	},
	false
);

let anchors = document.querySelectorAll('a');
for (let a of anchors) {
	a.addEventListener(
		'click',
		(e) => {
			sendBeacon({
				target: e.currentTarget.href,
				text: e.currentTarget.textContent,
				event: 'click',
			});
		},
		false
	);
}

function makeid() {
	let array = new Uint8Array(20);
	self.crypto.getRandomValues(array);
	return btoa(String.fromCharCode.apply(null, array));
}
