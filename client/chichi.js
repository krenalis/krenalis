let endpoint = 'https://localhost:2020/chichi.cgi/log-event';

function sendBeacon(data) {
	let d = data !== null ? data : {};

	// populate the data object with relevant infos.
	navigator.geolocation.getCurrentPosition(
		(position) => {
			d.geolocation = `latitude: ${position.coords.latitude} - longitude: ${position.coords.longitude}`;
		},
		(err) => {
			console.log(`cannot get the geolocation: ${err}`);
		}
	);
	d.referrer = document.referrer;
	d.connection = navigator.connection.type == undefined ? '' : navigator.connection.type;
	d.language = navigator.language;
	d.browser = navigator.userAgent;
	d.os = navigator.userAgentData.platform;
	d.isMobile = navigator.userAgentData.mobile;

	console.log(d);

	// serialize the data.
	let json;
	try {
		json = JSON.stringify(d);
	} catch (err) {
		console.error(`cannot parse ${d}`);
	}

	// send the JSON.
	navigator.sendBeacon(endpoint, json);
}

window.addEventListener(
	'load',
	(e) => {
		sendBeacon({});
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
			});
		},
		false
	);
}
