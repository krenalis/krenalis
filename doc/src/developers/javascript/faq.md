{% extends "/layouts/doc.html" %}
{% macro Title string %}FAQ{% end %}
{% Article %}

# FAQ

### Can I instantiate more than one instance of the JavaScript SDK?

Yes, importing the SDK in your project. No, with the snippet in the browser.

### There are limits to the size of the events?

Yes, 32KB. If an event, serialized to JSON, is longer than 32KB, it is discarded.

### How many events are sent with each HTTP request?

The SDK handles event transmission by grouping them into batches. Upon an event trigger, it waits for 20 milliseconds to collect any additional events before sending them all together in a single HTTP request. Each batch request is kept under 500KB in size.

If the browser is unloading the page, the SDK employs the `keepalive` option of the `fetch` call (or `sendBeacon` in Firefox, or `XHR` in IE11) to ensure event delivery. However, in this situation, the batch request size is capped at 64KB.

### How are multiple tabs handled?

Among the tabs, a leader is elected to send events to the server. The other tabs persist the events they collect in a queue, and the leader tab reads these queues to add them to its own queue and send them in order. If the leader tab is closed or becomes unresponsive, another leader is elected to continue sending the events.

The JavaScript SDK makes every effort to ensure that events are not lost or left unsent. It employs robust mechanisms to capture and transmit events reliably, minimizing the risk of data loss even in challenging network conditions.

### What happens if a tab is hidden or closed?

When the tab receives the `visibilitychange` or `pagehide` event from the browser, it immediately persists the event queue and sends the queued events using the `fetch` function with the `keepalive` option, or `sendBeacon` with Firefox, or `XHR` with IE11.

If there were more events queued than could be sent, the remaining events will be sent by a new leading tab or, if no other tabs are present, subsequently by the same tab.
