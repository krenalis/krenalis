{% extends "/layouts/doc.html" %}
{% macro Title string %}Anonymous Users Strategies{% end %}
{% Article %}

# Anonymous users strategies

The strategies for anonymous users determine how they are unified with each other and with non-anonymous users.

Consider the following scenario: a user interacts with a device anonymously during period **A**. Then, the user logs in and continues to interact, this time in a non-anonymous manner during period **B**. Subsequently, the user logs out and resumes interacting anonymously during period **C**.

In each of the three time periods, there will be one user, anonymous for periods A and C, and non-anonymous for period B. The strategy determines which user is unified with another.

### Conversion strategy

The Conversion strategy unifies the anonymous user from period A with the non-anonymous user from period B. This strategy allows all data collected during the initial anonymous navigation to be unified with the data of the non-anonymous user as soon as they log in. However, from logout onward, a new session is started, the Anonymous ID is changed, and consequently, the collected anonymous data is maintained in a separate anonymous user.

### Fusion strategy

The Fusion strategy unifies the anonymous user from period A with the non-anonymous user from period B. From logout onward, the collected anonymous data is maintained in a separate anonymous user. However, unlike the Conversion strategy, the session and the Anonymous ID remain the same.

### Isolation strategy

The Isolation strategy never unifies the users. Consequently, there will be two anonymous users and one non-anonymous user. These users have different sessions and different Anonymous IDs.

### Preservation strategy

The Preservation strategy unifies the anonymous user data before login and after logout, keeping it separate from the non-anonymous user who has logged in. Thus, the non-anonymous user has a different session and Anonymous ID than the unified anonymous user.

## Implement a strategy

To implement a specific strategy, you need to set the strategy option when loading a Meergo SDK. For example, with the [JavaScript SDK](/developers/javascript-sdk) in the browser:

```javascript
meergo.load(writeKey, endpoint, { strategy: 'Conversion' });
```

Then, use the [`identify`](../events/identify) call when the user logs in and the [`reset`](../developers/javascript-sdk/methods#reset) method when the user logs out. You can customize different strategies for various devices or situations based on your requirements. Refer to the SDK documentation for more details on how to implement these strategies in your application.

The default strategy, if the strategy option is not specified, is the "Conversion" strategy.
