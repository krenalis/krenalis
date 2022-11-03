import ConnectorInput from "./ConnectorInput/ConnectorInput";
import ConnectorSelect from "./ConnectorSelect/ConnectorSelect";
import ConnectorTextarea from "./ConnectorTextarea/ConnectorTextarea";
import ConnectorKeyValue from "./ConnectorKeyvalue/ConnectorKeyValue";
import ConnectorCheckbox from "./ConnectorCheckbox/ConnectorCheckbox";
import ConnectorColorPicker from "./ConnectorColorPicker/ConnectorColorPicker";
import ConnectorRadios from "./ConnectorRadios/ConnectorRadios";
import ConnectorRange from "./ConnectorRange/ConnectorRange";
import ConnectorSwitch from "./ConnectorSwitch/ConnectorSwitch";

export let renderConnectorComponent = (c, onChange, value) => {
    let component;
    switch (c.ComponentType) {
        case 'Input':
            if (c.Rows === 0 || c.Rows === 1) {
                component = <ConnectorInput 
                    name={c.Name}
                    value={value != null ? value : c.Value}
                    label={c.Label}
                    placeholder={c.Placeholder}
                    type={c.Type === '' ? 'text' : c.Type}
                    minlength={c.MinLength !== 0 && c.MinLength}
                    maxlength={c.MaxLength !== 0 && c.MaxLength}
                    passwordToggle={c.Type === 'password'}
                    onChange={onChange}
                />
            } else {
                component = <ConnectorTextarea
                    name={c.Name}
                    value={value != null ? value : c.Value}
                    label={c.Label}
                    placeholder={c.Placeholder}
                    rows={c.Rows}
                    minlength={c.MinLength !== 0 && c.MinLength}
                    maxlength={c.MaxLength !== 0 && c.MaxLength}
                    onChange={onChange}
                />
            }
            break;
        case 'Select':
            component = <ConnectorSelect 
                name={c.Name}
                value={value != null ? value : c.Value}
                label={c.Label}
                placeholder={c.Placeholder}
                options={c.Options}
                onChange={onChange}
            />
            break;
        case 'Switch':
            component =  <ConnectorSwitch
                name={c.Name}
                value={value != null ? value : c.Value}
                label={c.Label}
                onChange={onChange}
            />
            break;
        case 'Checkbox':
            component = <ConnectorCheckbox 
                name={c.Name}
                value={value != null ? value : c.Value}
                label={c.Label}
                onChange={onChange}
            />
            break;
        case 'ColorPicker':
            component = <ConnectorColorPicker
                name={c.Name}
                value={value != null ? value : c.Value}
                label={c.Label}
                onChange={onChange}
            />
            break;
        case 'Radios':
            component = <ConnectorRadios 
                name={c.Name}
                value={value != null ? value : c.Value}
                label={c.Label}
                options={c.Options}
                onChange={onChange}
            />
            break;
        case 'Range':
            component = <ConnectorRange
                name={c.Name}
                value={value != null ? value : c.Value}
                label={c.Label}
                min={c.Min}
                max={c.Max}
                step={c.Step}
                onChange={onChange}
            />
            break;
        case 'KeyValue':
            component = <ConnectorKeyValue 
                name={c.Name} 
                value={value != null ? value : c.Value}
                label={c.Label} 
                keyComponent={c.KeyComponent}
                keyLabel={c.KeyLabel}
                valueComponent={c.ValueComponent} 
                valueLabel={c.ValueLabel}
                onChange={onChange}
            />
            break;
        default:
            break;
    }
    return component
}
