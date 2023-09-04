!(function () {
	var a = (window.chichianalytics = window.chichianalytics || []);
	if (a.load) {
		window.console && console.error && console.error('The ChiChi snippet is included twice');
	}
	a.load = function (key, url) {
		a.key = key;
		a.url = url;
		var s = document.createElement('script');
		s.async = !0;
		s.type = 'module';
		s.src = '../dist/chichi.js';
		var c = document.getElementsByTagName('script')[0];
		c.parentNode.insertBefore(s, c);
	};
	var methods = [
		'alias',
		'endSession',
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
				a.push([name].concat(arguments));
				return a;
			};
		})(methods[i]);
	}
	a.load('123456789', 'https://localhost:9090/api/v1/batch');
	a.page();
})();
