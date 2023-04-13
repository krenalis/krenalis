import { useState, useEffect, useContext } from 'react';
import './ConnectionForm.css';
import ConnectorField from '../ConnectorFields/ConnectorField';
import { NotFoundError, UnprocessableError } from '../../api/errors';
import { AppContext } from '../../context/AppContext';
import statuses from '../../constants/statuses';
import * as icons from '../../constants/icons';
import { SettingsContext } from '../../context/SettingsContext';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionForm = ({ connection: c }) => {
	let [fields, setFields] = useState([]);
	let [actions, setActions] = useState([]);
	let [values, setValues] = useState(null);

	let { API, showError, showStatus, redirect } = useContext(AppContext);

	useEffect(() => {
		const fetchUI = async () => {
			let [ui, err] = await API.connections.ui(c.ID);
			if (err) {
				if (err instanceof NotFoundError) {
					redirect('/admin/connections');
					showStatus(statuses.connectionDoesNotExistAnymore);
					return;
				}
				if (err instanceof UnprocessableError) {
					if (err.code === 'EventNotExists') {
						// TODO(@Andrea): find a way to show the full error message
						// in the toast notification when the server is started with
						// the CHICHI_DEBUG_UI environment variable set to 'true'.
						console.error(
							`Unprocessable: connection does not implement the 'load' event in its ServeUI method`
						);
						showError('Unexpected error. Contact the administrator for more informations.');
					}
					return;
				}
				showError(err);
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
		let [ui, err] = await API.connections.uiEvent(c.ID, e, values);
		if (err) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'EventNotExists') {
					// TODO(@Andrea): find a way to show the full error message
					// in the toast notification when the server is started with
					// the CHICHI_DEBUG_UI environment variable set to 'true'.
					console.error(`Unprocessable: connection does not implement the ${e} event in its ServeUI method`);
					showError('Unexpected error. Contact the administrator for more informations.');
				}
				return;
			}
			showError(err);
			return;
		}
		if (e === 'save') {
			showStatus(statuses.connectionSaved);
			return;
		}
		if (ui.Alert != null) {
			showStatus([ui.Alert.Variant, icons.EXCLAMATION, ui.Alert.Message]);
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
