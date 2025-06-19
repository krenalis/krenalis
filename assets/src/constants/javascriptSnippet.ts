export const SNIPPET = `<script>
	(function(){var e=window.meergo=window.meergo||[];if(e.load)window.console&&console.error&&console.error("The Meergo snippet is included twice");else{e.load=function(r,a,c){e.key=r,e.url=a,e.options=c;var o=document.createElement("script");o.async=!0,o.type="text/javascript",o.src="javaScriptSDKURL";var t=document.getElementsByTagName("script")[0];t.parentNode.insertBefore(o,t)};for(var s=["alias","close","debug","endSession","getAnonymousId","getSessionId","group","identify","page","ready","reset","screen","setAnonymousId","startSession","track","user"],n=0;n<s.length;n++)(function(r){e[r]=function(){return e.push([r].concat(Array.prototype.slice.call(arguments))),e}})(s[n]);
	meergo.load("writekey","endpoint");
	}})();
</script>`;
