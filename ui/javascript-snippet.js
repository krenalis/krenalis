// Minimize this code snippet by running: deno task minify-javascript-snippet
var analytics = window.chichiAnalytics = window.chichiAnalytics || [];
if (analytics.load) {
  window.console && console.error &&
    console.error("The ChiChi snippet is included twice");
} else {
  analytics.load = function (key, url, options) {
    analytics.key = key;
    analytics.url = url;
    analytics.options = options;
    var script = document.createElement("script");
    script.async = !0;
    script.type = "text/javascript";
    script.src = "/javascript-sdk/dist/chichi.min.js";
    var elem = document.getElementsByTagName("script")[0];
    elem.parentNode.insertBefore(script, elem);
  };
  var methods = [
    "alias",
    "anonymize",
    "close",
    "debug",
    "endSession",
    "getAnonymousId",
    "getSessionId",
    "group",
    "identify",
    "page",
    "ready",
    "reset",
    "screen",
    "setAnonymousId",
    "startSession",
    "track",
    "user",
  ];
  for (var i = 0; i < methods.length; i++) {
    (function (name) {
      analytics[name] = function () {
        analytics.push([name].concat(Array.prototype.slice.call(arguments)));
        return analytics;
      };
    })(methods[i]);
  }
  chichiAnalytics.load("writekey", "endpoint");
}