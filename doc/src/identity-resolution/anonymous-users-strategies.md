# Anonymous Users Strategies

The anonymous users strategies determine how anonymous users are unified with each other and with non-anonymous users.

Let's consider the following scenario. A user interacts with a device anonymously; let's call this period of time **A**. Then, the user logs in and continues to interact, but this time in a non-anonymous manner; let's call this period **B**. Afterward, the user logs out and continues to interact, but again in an anonymous manner; let's call this period **C**.

There will be one user in each of the three time periods, anonymous for A and C, and non-anonymous for B. The strategy determines which user will be unified with another.

### AB-C Strategy

The AB-C strategy unifies the anonymous user from A with the non-anonymous user from B. This strategy allows all data collected during the initial anonymous navigation to be unified with the data of the non-anonymous user as soon as they log in. From logout onward, however, the collected anonymous data is maintained in a separate anonymous user.

### ABC Strategy

The ABC strategy unifies the anonymous users from A and C with the non-anonymous user from B. This strategy allows all data collected during anonymous navigation before login and after logout to be unified with the data of the non-anonymous user.

### A-B-C Strategy

The A-B-C strategy never unifies the users. Consequently, there will be two anonymous users and one non-anonymous user.

### AC-B Strategy

The AC-B strategy unifies the anonymous user data before login and after logout, keeping it separate from the non-anonymous user who has logged in.

## Implement a Strategy

To implement a specific strategy, you need to set the strategy option when loading a Chichi SDK. For example, with the [JavaScript SDK](../javascript-sdk.md) in the browser:

```javascript
chichianalytics.load(writeKey, endpoint, { strategy: 'AB-C' });
```

Then, use the [`identify`](../events/identify.md) call when the user logs in and the [`anonymize`](../events/anonymize.md) call when the user logs out. You can customize different strategies for various devices or situations based on your requirements. Refer to the SDK documentation for more details on how to implement these strategies in your application.

The default strategy, if the strategy option is not specified, is the "AB-C" strategy.