{% extends "/layouts/doc.html" %}
{% macro Title string %}Settings and UI{% end %}
{% Article %}

# Settings and UI

## Settings

A connector typically needs to maintain settings to operate. For example, the Google Analytics connector needs to store values for "Measurement ID" and "API Secret," so a specific data type is defined:

```go
type Settings struct {
    MeasurementID string
    APISecret     string
}
```

Defining a specific type for settings is convenient because settings are exchanged in JSON format between the connector and Meergo. Therefore, having a dedicated data structure allows for easier serialization and deserialization into JSON. For instance, the settings for an instance of the Google Analytics connector could be serialized like this:

```json
{"MeasurementID":"G-2XYZBEB6AB","APISecret":"ZuHCHFZbRBi8V7u8crWFUz"}
```

Meergo does not impose any constraints on how settings are serialized, except that they must be in JSON format.

When Meergo creates an instance of a connector, it passes the current settings through the `Settings` field and a function through the `SetSettings` field, which the connector can use to update the settings. For example, the constructor of the Google Analytics connector stores the configuration passed as an argument and the `SetSettings` function, and deserializes the settings:

```go
func New(env *meergo.AppEnv) (*Analytics, error) {
    c := Analytics{env: env}
    if len(env.Settings) > 0 {
        err := json.Unmarshal(env.Settings, &c.settings)
        if err != nil {
            return nil, errors.New("cannot unmarshal settings of Google Analytics connector")
        }
    }
    return &c, nil
}
```

Later, when it needs to update the settings, it serializes them and calls the `SetSettings` function:

```go
...
settings, err := json.Marshal(s.settings)
if err != nil {
    return err
}
err = ga.env.SetSettings(ctx, settings)
if err != nil {
    return err
}
...
```

## User interface

If the connector requires all or part of the settings to be configurable by the user, it can define a user interface (UI) that will be integrated into Meergo's admin. Various components are available to create the interface for a connector: `Input`, `Select`, `Checkbox`, `ColorPicker`, `Radios`, `Range`, `Switch`, `KeyValue`, `FieldSet`, `AlternativeFieldSets`, `Text`, `Button`, and `Alert`.

For example, for the previous two settings of Google Analytics, the interface could be defined as follows:

```go
&meergo.UI{
    Fields: []meergo.Component{
        &meergo.Input{
            Name:        "MeasurementID",
            Label:       "Measurement ID",
            Placeholder: "G-2XYZBEB6AB",
            Type:        "text",
            MinLength:   2,
            MaxLength:   20,
            HelpText:    "Follow these instructions to get your Measurement ID: https://support.google.com/analytics/answer/9539598#find-G-ID",
        },
        &meergo.Input{
            Name:        "APISecret",
            Label:       "API Secret",
            Placeholder: "ZuHCHFZbRBi8V7u8crWFUz",
            Type:        "text",
            MinLength:   1,
            MaxLength:   40,
        },
    },
    Settings: settings,
}
```

Two `Input` components are present, "MeasurementID" and "APISecret," and a `Button` component with an event named "save." A `Button` component with the "save" event must always be present, as it ensures to Meergo that the options of the interface fields have been saved in the connector's settings.

The `Settings` field, of type `json.Value`, contains the settings of the interface components in JSON format. For example, `Settings` could have the following value:

```go
json.Value(`{"MeasurementID":"G-2XYZBEB6AB","APISecret":"ZuHCHFZbRBi8V7u8crWFUz"}`)
```

### Serving the UI

To provide the interface to the user and respond to events triggered by user interaction, the connector must implement the `ServeUI` method and eventually declare in its configuration that it has settings (see the documentation specific for the connector type):

```go
ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error)
```

- `ctx`: Context, it's never `nil`.
- `event`: Event to serve.
- `settings`: Settings of the interface components, serialized in JSON. If not `nil`, it's always valid JSON.
- `role`: Connection's role, it can be `Source` or `Destination`.

The `ServeUI` method must serve the `"load"` and `"save"` events:

- `"load"`: Initially, when the connector's interface is loaded, `ServeUI` is called with the `"load"` event and `settings` set to `nil`. If no errors occur, the method must return a non-`nil` interface.
- `"save"`: When the user clicks the "Add" or "Save" button, `ServeUI` is called with the `"save"` event and `settings` representing the user-entered settings serialized in JSON. The method should validate and save the settings and return a `nil` interface.

It must also serve any other event associated with the buttons present in the interface. When called for one of these events, `settings` are in JSON-serialized format of the user-entered settings. If the method returns a non-`nil` interface, the UI is updated with the returned interface.

If the event is not among those expected, the method should return the `ErrUIEventNotExist` error. If the settings passed as arguments are not valid, it should return an error of type `InvalidSettingsError`. You can use the `meergo.NewInvalidSettingsError` function for this purpose.

#### Example

The following is the `ServeUI` method of the Google Analytics connector:

```go
func (ga *Analytics) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

    switch event {
    case "load":
        // Load the user interface.
        var s settings
        if ga.settings != nil {
            s = *ga.settings
        }
        settings, _ = json.Marshal(s)
    case "save":
        // Validate and save the settings.
        s, err := validateSettings(settings)
        if err != nil {
            return nil, err
        }
        return nil, ga.env.SetSettings(ctx, s)
    default:
        return nil, meergo.ErrEventNotExist
    }

    ui := &meergo.UI{
        Fields: []meergo.Component{
            &meergo.Input{Name: "MeasurementID", Label: "Measurement ID", Placeholder: "G-2XYZBEB6AB", Type: "text", MinLength: 2, MaxLength: 20, HelpText: "Follow these instructions to get your Measurement ID: https://support.google.com/analytics/answer/9539598#find-G-ID"},
            &meergo.Input{Name: "APISecret", Label: "API Secret", Placeholder: "ZuHCHFZbRBi8V7u8crWFUz", Type: "text", MinLength: 1, MaxLength: 40},
        },
        Settings: settings,
    }

    return ui, nil
}
```

Pay attention to the following:

- If `event` is `"load"`, it returns the user interface.
- If `event` is `"save"`, it validates and saves the settings of the interface fields in the settings using the `SetSettings` function, and returns a `nil` interface.
- Before saving, the settings are validated through its own method, which returns an `InvalidSettingsError` error if the settings do not pass validation.
- If `event` is unknown, it returns the `ErrUIEventNotExist` error.
- The serialized JSON settings are directly assigned to the `Settings` field of the returned interface. This is because in this case, each interface field is associated with a setting with the same name as the field.
