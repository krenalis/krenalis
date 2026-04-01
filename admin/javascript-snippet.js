// Minimize this code snippet by running: npm run minify-snippet
var e = window.krenalis = window.krenalis || [];
if (e.load) {
  window.console && console.error &&
    console.error("The Krenalis snippet is included twice");
} else {
  e.load = function (key, url, options) {
    e.key = key;
    e.url = url;
    e.options = options;
    var script = document.createElement("script");
    script.async = !0;
    script.type = "text/javascript";
    script.src = "javaScriptSDKURL";
    var elem = document.getElementsByTagName("script")[0];
    elem.parentNode.insertBefore(script, elem);
  };
  var methods = [
    "alias",
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
      e[name] = function () {
        e.push([name].concat(Array.prototype.slice.call(arguments)));
        return e;
      };
    })(methods[i]);
  }
  krenalis.load("writekey", "endpoint");
}