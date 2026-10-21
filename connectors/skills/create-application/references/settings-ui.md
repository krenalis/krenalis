# Settings UI

Implement the following method if and only if at least one declared role has `HasSettings` set. Set the flag only on roles that expose configuration UI:

```go
ServeUI(context.Context, string, json.Value, connectors.Role) (*connectors.UI, error)
```

Registry validation rejects both a missing method for a true flag and a method when neither role enables settings.

## Persist settings through the store

Use `ApplicationEnv.Settings.Load` and `Store`. The current environment does not expose settings as a raw JSON field and has no `SetSettings` method.

Define an internal settings struct with stable JSON names. A conventional flow is:

- `load`: load persisted settings, marshal them for `UI.Settings`, and return the form;
- `save`: decode the supplied `json.Value`, validate it, optionally perform a bounded provider check, store it, and return;
- custom event: handle it explicitly or return `connectors.ErrUIEventNotExist`.

Return `connectors.NewInvalidSettingsError` or `connectors.NewInvalidSettingsErrorf` for correctable user input. Return operational errors normally. Do not persist partially validated settings.

## Build the form

Use concrete components from `connectors/ui.go`, such as `Input`, `Select`, `Checkbox`, `Radios`, `Switch`, `KeyValue`, or `AlternativeFieldSets`. Use `connectors.SaveButton` for persistence. Set component `Role` when a shared UI contains role-specific fields.

Keep UI settings and the persisted struct round-trippable. Preserve unknown or secret values only when the product workflow requires it and tests demonstrate the behavior. Do not assume that an empty submitted secret means “keep the old value” unless the current UI lifecycle supports that distinction and the connector tests creation, replacement, deliberate clearing, and role changes. Never echo a credential into alerts or validation messages.

Settings and UI complexity are capability-specific. Do not add settings merely because nearby connectors have them, and do not make a remote validation request a universal save requirement.

## Test the lifecycle

Cover load with empty and existing data, successful save, invalid JSON/type/length values, role-specific controls, custom events, store failures, and provider-check failures when present. Assert that failed validation does not store data and that secret values do not appear in returned errors or UI alerts.
