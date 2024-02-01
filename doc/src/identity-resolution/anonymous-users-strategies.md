# Anonymous Users Strategies

The anonymous users strategies determine how anonymous users are unified with each other and with non-anonymous users.

Let's consider the following scenario. A user interacts with a device anonymously; let's call this period of time **A**. Then, the user logs in and continues to interact, but this time in a non-anonymous manner; let's call this period **B**. Afterward, the user logs out and continues to interact, but again in an anonymous manner; let's call this period **C**.

There will be one user in each of the three time periods, anonymous for A and C, and non-anonymous for B. The strategy determines which user will be unified with another.

### AB-C Strategy

The AB-C strategy unifies the anonymous user from A with the non-anonymous user from B. This strategy allows all data collected during the initial anonymous navigation to be unified with the data of the non-anonymous user as soon as they log in. From logout onward, however, the collected anonymous data is maintained in a separate anonymous user.

To implement this strategy, when the user logs out, the `reset` method is called.

### ABC Strategy

The ABC strategy unifies the anonymous users from A and C with the non-anonymous user from B. This strategy allows all data collected during anonymous navigation before login and after logout to be unified with the data of the non-anonymous user.

To implement this strategy, the `reset` method is never called.

### A-B-C Strategy

The A-B-C strategy never unifies the users. Consequently, there will be two anonymous users and one non-anonymous user.

To implement this strategy, when the user logs in and logs out, the `reset` method is called.

### AC-B Strategy

The AC-B strategy unifies the anonymous user data before login and after logout, keeping it separate from the non-anonymous user who has logged in.

To implement this strategy:

* Call the `getAnonymousId` method and save the returned Anonymous ID on the device.
* When the user logs in, call the `reset` method.
* When the user logs out, call the `reset` method, then call the `setAnonymousId` method with the previously saved Anonymous ID as argument.