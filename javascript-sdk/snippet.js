!(function () {
	var a = (window.chichiAnalytics = window.chichiAnalytics || []);
	if (a.load) {
		window.console && console.error && console.error('The ChiChi snippet is included twice');
		return;
	}
	a.load = function (key, url, options) {
		a.key = key;
		a.url = url;
		a.options = options;
		var s = document.createElement('script');
		s.async = !0;
		s.type = 'text/javascript';
		s.src = '/javascript-sdk/dist/chichi.min.js';
		var c = document.getElementsByTagName('script')[0];
		c.parentNode.insertBefore(s, c);
	};
	var methods = [
		'alias',
		'anonymize',
		'debug',
		'endSession',
		'getAnonymousId',
		'getSessionId',
		'group',
		'identify',
		'page',
		'ready',
		'reset',
		'screen',
		'setAnonymousId',
		'startSession',
		'track',
		'user',
	];
	for (var i = 0; i < methods.length; i++) {
		(function (name) {
			a[name] = function () {
				a.push([name].concat(Array.prototype.slice.call(arguments)));
				return a;
			};
		})(methods[i]);
	}
	a.load('kxe7WIDDGvcfDEKgHePfHzuHQ6dTU2xc', 'https://localhost:9090/api/v1/');
})();
