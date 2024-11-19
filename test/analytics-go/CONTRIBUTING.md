## Running Tests

From the root of this repository, run the command:

```console
go test  ./...
```

## New Version Release Procedure

1. Update the value for the `Version` constant inside the `analytics.go` file.
2. Update the tests inside `fixtures`.
3. Ensure that the tests pass. See the specific section in this file.
4. Release a new version by tagging the repository.
