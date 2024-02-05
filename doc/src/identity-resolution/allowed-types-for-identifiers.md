# Allowed Types For Identifiers

Here is a list of allowed types for the [identifiers](workspace-identity-resolution.md#identifiers) used in the [Workspace Identity Resolution](workspace-identity-resolution.md):

* `Int(n)`       
* `UInt(n)`      
* `Decimal(p,s)`, but only if scale `s` is 0
* `UUID`         
* `Inet`         
* `Text`         

> Note that meta properties cannot be used as identifiers. This should be
> documented in a consistent way.