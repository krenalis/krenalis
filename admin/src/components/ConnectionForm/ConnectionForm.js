import { useState, useEffect } from 'react';
import './ConnectionForm.css';
import ConnectorField from '../ConnectorFields/ConnectorField';
import call from '../../utils/call';
import { SettingsContext } from '../../context/SettingsContext';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionForm = ({ connection: c, onStatusChange, onError }) => {
	let [fields, setFields] = useState([]);
	let [actions, setActions] = useState([]);
	let [values, setValues] = useState(null);

	useEffect(() => {
		const fetchUI = async () => {
			let [ui, err] = await call('/admin/connections/ui', 'POST', c.ID);
			if (err) {
				onError(err);
				return;
			}
			setFields(ui.Form.Fields);
			setActions(ui.Form.Actions);
			setValues(ui.Form.Values);
		};
		fetchUI();
	}, []);

	const onActionClick = async (e) => {
		// remove the errors
		let fls = [];
		for (let f of fields) {
			f.Error = '';
			fls.push(f);
		}
		setFields(fls);

		let [ui, err] = await call('/admin/connections/ui-event', 'POST', {
			connection: c.ID,
			event: e,
			values: values,
		});
		if (err != null) {
			onError(err);
			return;
		}
		if (ui.Alert != null) {
			onStatusChange({ variant: ui.Alert.Variant, icon: 'exclamation-square', text: ui.Alert.Message });
		}
		if (ui.Form != null) {
			setFields(ui.Form.Fields);
			setActions(ui.Form.Actions);
			setValues(ui.Form.Values);
		}
	};

	const onFieldChange = (name, value) => {
		setValues((prevValues) => ({ ...prevValues, [name]: value }));
	};

	let fieldsToRender = [];
	for (let f of fields) {
		fieldsToRender.push(<ConnectorField field={f} />);
	}

	let actionsToRender = [];
	for (let a of actions) {
		actionsToRender.push(
			<SlButton
				variant={a.Variant}
				onClick={async () => {
					await onActionClick(a.Event);
				}}
			>
				{a.Text}
			</SlButton>
		);
	}

	return (
		<div className='form'>
			<SettingsContext.Provider value={{ values: values, onChange: onFieldChange }}>
				<div className='fields'>{fieldsToRender}</div>
			</SettingsContext.Provider>
			<div className='actions'>{actionsToRender}</div>
		</div>
	);
};

export default ConnectionForm;
