# Settings + UI (ServeUI)

Settings are stored as JSON. Krenalis provides:

- `env.Settings` (current settings JSON)
- `env.SetSettings(ctx, newSettingsJSON)` to persist updates

If the connector needs user configuration in the Admin console, implement:

```go
ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error)
```

Where `json.Value` is `github.com/krenalis/krenalis/tools/json.Value` (not the standard library `encoding/json` package).

Rules:

- Must handle `"load"`:
  - `settings` is nil on first load
  - return a non-nil UI with fields, initial settings JSON, **and at least one button**
  - the typical case is `Buttons: []connectors.Button{connectors.SaveButton}`
- Must handle `"save"`:
  - validate settings
  - persist via `env.SetSettings(...)`
  - return `(nil, err)` on save
  - update the connector instance's in-memory settings too (so subsequent calls use the newly saved settings)
- Unknown events: return `connectors.ErrUIEventNotExist`
- For invalid settings: return `connectors.NewInvalidSettingsError(...)` (or `...Errorf`)
- Do not add redundant UTF-8 checks for string settings (Krenalis already guarantees UTF-8 strings).
- For security-sensitive settings (API keys/tokens/secrets/base URLs), make validation complete:
  - enforce both min and max length for secrets/tokens (no "min only" checks)
  - validate provider-required format/prefix/allowed charset when documented
  - enforce strict URL validation for base URLs
  - reject oversized values explicitly

**Buttons** — every UI response must include at least one button in `Buttons []connectors.Button`.
Use `connectors.SaveButton` for the standard save action (its event is `"save"`, which is the special event that triggers settings persistence):

```go
return &connectors.UI{
    Fields:   fields,
    Settings: settings,
    Buttons:  []connectors.Button{connectors.SaveButton},
}, nil
```

Use available UI components in `connectors/ui.go`:

- `Input`, `Select`, `Checkbox`, `Radios`, `Switch`, `Range`, `KeyValue`, `FieldSet`, `AlternativeFieldSets`, `Text`, etc.

Important:

- If you set `HasSettings: true` in spec, you must implement `ServeUI`.
- Avoid leaking secrets in UI. Validate and store them, but never echo them in a preview request.
