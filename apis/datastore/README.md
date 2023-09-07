# apis/datastore

- [Redis indexes example](#redis-indexes-example)
  - [Arrays](#arrays)


## Redis indexes example

Property-values pairs are stored as:

| Key                           | Value | Type   |
|-------------------------------|-------|--------|
| property:firstname:"John"     | "1"   | string |
| property:firstname:"Paul"     | "2"   | string |
| property:lastname:"Lennon"    | "1"   | string |
| property:lastname:"McCartney" | "2"   | string |
| property:band:"The Beatles"   | "1,2" | string |

While user properties are stored as:

| Key    | Value                                                                    | Type   |
|--------|--------------------------------------------------------------------------|--------|
| user:1 | "{"id":1,"firstname":"John","lastname":"Lennon","band":"The Beatles}"    | string |
| user:2 | "{"id":2,"firstname":"Paul","lastname":"McCartney","band":"The Beatles}" | string |

### Arrays

Let's suppose that we have an user with GID 12 with a single property "phone_numbers", which holds two different values, `333` and `444`. That user is stored in Redis as:

| Key                        | Value                                     | Type   |
| -------------------------- | ----------------------------------------- | ------ |
| property:phone_numbers:333 | "12"                                      | string |
| property:phone_numbers:444 | "12"                                      | string |
| user:12                    | "{"id":12,"phone_numbers":["333","444"]}" | string |

