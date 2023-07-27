import { useState, useEffect, useContext, useRef } from 'react';
import ConnectorField from '../../shared/ConnectorFields/ConnectorField';
import ConfirmationButton from '../../shared/ConfirmationButton/ConfirmationButton';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { AppContext } from '../../../context/providers/AppProvider';
import statuses from '../../../constants/statuses';
import * as icons from '../../../constants/icons';
import { SettingsContext } from '../../../context/SettingsContext';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

const Form = ({ connection: c }) => {
	const [fields, setFields] = useState([]);
	const [actions, setActions] = useState([]);
	const [values, setValues] = useState(null);

	const { api, showError, showStatus, redirect } = useContext(AppContext);

	const confirmationButtonsRef = useRef([]);

	useEffect(() => {
		const fetchUI = async () => {
			const [ui, err] = await api.connections.ui(c.id);
			if (err) {
				if (err instanceof NotFoundError) {
					redirect('connections');
					showStatus(statuses.connectionDoesNotExistAnymore);
					return;
				}
				if (err instanceof UnprocessableError) {
					if (err.code === 'EventNotExists') {
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
		const confirmationButton = confirmationButtonsRef.current[confirmationButtonIndex];

		// remove the errors
		const fls = [];
		for (const f of fields) {
			f.Error = '';
			fls.push(f);
		}
		setFields(fls);
		if (confirmationButton != null) {
			confirmationButton.load();
		}
		const [ui, err] = await api.connections.uiEvent(c.id, eventName, values);
		if (confirmationButton != null) {
			confirmationButton.stop();
		}
		if (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'EventNotExists') {
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

	const fieldsToRender = [];
	for (const f of fields) {
		fieldsToRender.push(<ConnectorField field={f} />);
	}

	const actionsToRender = [];
	for (const [i, a] of actions.entries()) {
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

export default Form;
