# apis/datastore

- [Redis indexes](#redis-indexes)


## Redis indexes

On Redis we keep these indexes:

| Key                                 | Value               | Note                                             |
|-------------------------------------|---------------------|--------------------------------------------------|
| `props:<property>:<property value>` | `[<user GID>, ...]` | For properties with non-zero values              |
| `props:<property>:-`                | `[<user GID>, ...]` | For properties with zero values                  |
| `user_prop_keys:<user GID>`         | `[<key 1>, ...]`    | For holding the keys of the properties of a user |