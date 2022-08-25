let endpoint = 'https://localhost:9090/log-event';

function sendBeacon(data) {
	let d = data == null ? {} : data;

	// populate the data object with relevant infos.
	let session = localStorage.getItem('chichi-session');
	if (session == null) {
		session = makeid(8);
		localStorage.setItem('chichi-session', session);
	}
	d.session = session;
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
	let date = new Date();
	d.timestamp = date.toISOString();
	d.referrer = document.referrer;
	if (navigator.connection) {
		d.connection = navigator.connection.type == null ? '' : navigator.connection.type;
	}
	d.language = navigator.language;
	d.browser = navigator.userAgent;
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
			event: 'visit',
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

function makeid(length) {
	let result = '';
	let characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
	let charactersLength = characters.length;
	for (let i = 0; i < length; i++) {
		result += characters.charAt(Math.floor(Math.random() * charactersLength));
	}
	return result;
}
