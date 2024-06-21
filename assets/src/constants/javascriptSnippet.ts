export const SNIPPET = `<script>
	(function(){var e=window.chichiAnalytics=window.chichiAnalytics||[];if(e.load)window.console&&console.error&&console.error("The ChiChi snippet is included twice");else{e.load=function(n,r,o){e.key=n,e.url=r,e.options=o;var t=document.createElement("script");t.async=!0,t.type="text/javascript",t.src="/javascript-sdk/dist/chichi.min.js";var s=document.getElementsByTagName("script")[0];s.parentNode.insertBefore(t,s)};for(var c=["alias","close","debug","endSession","getAnonymousId","getSessionId","group","identify","page","ready","reset","screen","setAnonymousId","startSession","track","user"],i=0;i<c.length;i++)(function(n){e[n]=function(){return e.push([n].concat(Array.prototype.slice.call(arguments))),e}})(c[i]);
	chichiAnalytics.load("writekey","endpoint");
	}})();
</script>`;
