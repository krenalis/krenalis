# Settings + UI (ServeUI)

Settings are persisted by Krenalis and exposed to connectors through
`env.Settings`, a `connectors.SettingsStore`:

```go
type SettingsStore interface {
    Load(ctx context.Context, dst any) error
    Store(ctx context.Context, src any) error
}
```

`Load` reads current persisted settings into a Go value. `Store` persists a
validated Go value. In normal connector code, pass the settings struct directly
to `Store` (do not marshal it to JSON first).

If the connector needs user configuration in the Admin console, implement:

```go
ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error)
```

Where `json.Value` is `github.com/krenalis/krenalis/tools/json.Value` (not the standard library `encoding/json` package).

Important distinction:

- `env.Settings` is the persisted settings store for the connector.
- The `settings json.Value` parameter passed to `ServeUI` is the Admin UI payload for the current event.
- On `"load"`, read persisted settings with `env.Settings.Load(ctx, &s)` and return them in `UI.Settings`.
- On `"save"`, validate the `settings` payload, then persist with `env.Settings.Store(ctx, s)`.

## Canonical pattern

Constructor:

```go
func New(env *connectors.ApplicationEnv) (*Example, error) {
    return &Example{env: env}, nil
}

type Example struct {
    env *connectors.ApplicationEnv
}
```

`ServeUI` + save:

```go
func (c *Example) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {
    switch event {
    case "load":
        var s innerSettings
        if err := c.env.Settings.Load(ctx, &s); err != nil {
            return nil, err
        }
        settings, _ = json.Marshal(s)
    case "save":
        return nil, c.saveSettings(ctx, settings)
    default:
        return nil, connectors.ErrUIEventNotExist
    }

    return &connectors.UI{
        Fields:   fields,
        Settings: settings,
        Buttons:  []connectors.Button{connectors.SaveButton},
    }, nil
}
func (c *Example) saveSettings(ctx context.Context, settings json.Value) error {
    var s innerSettings
    if err := settings.Unmarshal(&s); err != nil {
        return err
    }

    // Validate s completely before storing it.

    return c.env.Settings.Store(ctx, s)
}
```

Operational methods:

```go
var s innerSettings
if err := c.env.Settings.Load(ctx, &s); err != nil {
    return err
}
```

Use this pattern in methods such as `RecordSchema`, `Records`, `Upsert`,
`EventTypes`, `SendEvents`, `PreviewSendEvents`, and helper methods that build
authenticated API requests.

## Rules

- Must handle `"load"`:
  - the `settings` parameter is normally nil on first load and is not the persisted state
  - read persisted settings through `env.Settings.Load(ctx, &s)`
  - apply any local defaults after `Load` if zero values need UI defaults
  - return a non-nil UI with fields, initial settings JSON, **and at least one button**
  - use `Buttons: []connectors.Button{connectors.SaveButton}` in the normal case
- Must handle `"save"`:
  - unmarshal and validate the `settings` payload received by `ServeUI`
  - persist via `env.Settings.Store(ctx, s)` after validation succeeds
  - always return a nil `*UI` on `"save"`, regardless of outcome: `(nil, nil)` on success and `(nil, err)` on validation/persistence failure
- Unknown events: return `connectors.ErrUIEventNotExist`
- For invalid settings: return `connectors.NewInvalidSettingsError(...)` (or `...Errorf`)
- Do not add redundant UTF-8 checks for string settings (Krenalis already guarantees UTF-8 strings).
- For security-sensitive settings (API keys/tokens/secrets/base URLs), make validation complete:
  - enforce both min and max length for secrets/tokens (no "min only" checks)
  - validate provider-required format/prefix/allowed charset when documented
  - enforce strict URL validation for base URLs
  - reject oversized values explicitly

## Do not

- Do not read `env.Settings` as `json.Value` (`env.Settings.Unmarshal`, `len(env.Settings)`, etc.).
- Do not use `env.SetSettings`; it is the old API.
- Do not marshal the settings struct before the normal `env.Settings.Store(ctx, s)` call.
- Do not keep settings in connector instance fields (for example `c.settings`).
- Do not load persisted settings in `New`.

Use available UI components in `connectors/ui.go`:

- `Input`, `Select`, `Checkbox`, `Radios`, `Switch`, `Range`, `KeyValue`, `FieldSet`, `AlternativeFieldSets`, `Text`, etc.

Important:

- If you set `HasSettings: true` in spec, you must implement `ServeUI`.
- Avoid leaking secrets in UI. Validate and store them, but never echo them in a preview request.
