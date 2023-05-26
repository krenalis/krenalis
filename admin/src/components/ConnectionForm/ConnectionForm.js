import { useState, useEffect, useContext, useRef } from 'react';
import './ConnectionForm.css';
import ConnectorField from '../ConnectorFields/ConnectorField';
import ConfirmationButton from '../ConfirmationButton/ConfirmationButton';
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

	const confirmationButtonsRef = useRef([]);

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

	const onActionClick = async (eventName, confirmationButtonIndex) => {
		let confirmationButton = confirmationButtonsRef.current[confirmationButtonIndex];

		// remove the errors
		let fls = [];
		for (let f of fields) {
			f.Error = '';
			fls.push(f);
		}
		setFields(fls);
		if (confirmationButton != null) {
			confirmationButton.load();
		}
		let [ui, err] = await API.connections.uiEvent(c.ID, eventName, values);
		if (confirmationButton != null) {
			confirmationButton.stop();
		}
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
						`Unprocessable: connection does not implement the ${eventName} event in its ServeUI method`
					);
					showError('Unexpected error. Contact the administrator for more informations.');
				}
				return;
			}
			showError(err);
			return;
		}
		if (eventName === 'save') {
			showStatus(statuses.connectionSaved);
			return;
		}
		if (ui == null) {
			if (confirmationButton != null) {
				confirmationButton.confirm();
			}
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
	for (let [i, a] of actions.entries()) {
		if (a.Confirm) {
			actionsToRender.push(
				<ConfirmationButton
					variant={a.Variant}
					onClick={async () => {
						await onActionClick(a.Event, i);
					}}
					ref={(ref) => {
						confirmationButtonsRef.current[i] = ref;
					}}
				>
					{a.Text}
				</ConfirmationButton>
			);
		} else {
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
