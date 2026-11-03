export const SNIPPET = `<script>
  (function(){var e=window.krenalis=window.krenalis||[];if(e.load)window.console&&console.error&&console.error("The Krenalis snippet is included twice");else{e.load=function(n,a,i){e.key=n,e.url=a,e.options=i;var r=document.createElement("script");r.async=!0,r.type="text/javascript",r.src="javaScriptSDKURL";var t=document.getElementsByTagName("script")[0];t.parentNode.insertBefore(r,t)};for(var o=["alias","close","debug","endSession","getAnonymousId","getSessionId","group","identify","page","ready","reset","screen","setAnonymousId","setConsents","startSession","track","user"],s=0;s<o.length;s++)(function(n){e[n]=function(){return e.push([n].concat(Array.prototype.slice.call(arguments))),e}})(o[s]);
  krenalis.load("writekey","endpoint");
  }})();
</script>`;

export const DOCUMENTATION_LINK = 'https://www.krenalis.com/docs/ref/admin/javascript-sdk';