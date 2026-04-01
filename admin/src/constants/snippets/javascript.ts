export const SNIPPET = `<script>
  (function(){var e=window.krenalis=window.krenalis||[];if(e.load)window.console&&console.error&&console.error("The Krenalis snippet is included twice");else{e.load=function(n,a,i){e.key=n,e.url=a,e.options=i;var r=document.createElement("script");r.async=!0,r.type="text/javascript",r.src="javaScriptSDKURL";var o=document.getElementsByTagName("script")[0];o.parentNode.insertBefore(r,o)};for(var s=["alias","close","debug","endSession","getAnonymousId","getSessionId","group","identify","page","ready","reset","screen","setAnonymousId","startSession","track","user"],t=0;t<s.length;t++)(function(n){e[n]=function(){return e.push([n].concat(Array.prototype.slice.call(arguments))),e}})(s[t]);
  krenalis.load("writekey","endpoint");
  }})();
</script>`;

export const DOCUMENTATION_LINK = 'https://www.krenalis.com/docs/ref/admin/javascript-sdk';