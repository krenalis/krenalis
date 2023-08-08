# apis/datastore

- [Redis indexes example](#redis-indexes-example)


## Redis indexes example

Identifier-values pairs are stored as:

| Key                           | Value | Type   |
|-------------------------------|-------|--------|
| property:firstname:"John"     | "1"   | string |
| property:firstname:"Paul"     | "2"   | string |
| property:lastname:"Lennon"    | "1"   | string |
| property:lastname:"McCartney" | "2"   | string |
| property:band:"The Beatles"   | "1,2" | string |

While user identifiers are stored as:

| Key    | Value                                                                    | Type   |
|--------|--------------------------------------------------------------------------|--------|
| user:1 | "{"id":1,"firstname":"John","lastname":"Lennon","band":"The Beatles}"    | string |
| user:2 | "{"id":2,"firstname":"Paul","lastname":"McCartney","band":"The Beatles}" | string |


