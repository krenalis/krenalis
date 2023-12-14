import React, { useState, useEffect, useContext, useRef, ReactNode } from 'react';
import ConnectorField from '../../shared/ConnectorFields/ConnectorField';
import FeedbackButton, { FeedbackButtonRef } from '../../shared/FeedbackButton/FeedbackButton';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import AppContext from '../../../context/AppContext';
import statuses from '../../../constants/statuses';
import * as icons from '../../../constants/icons';
import SettingsForm from '../../shared/SettingsForm/SettingsForm';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import { UIResponse, UIValues } from '../../../types/external/api';
import ConnectorFieldInterface, { ConnectorAction } from '../../../types/external/ui';

interface FormProps {
	connection: TransformedConnection;
}

const ConnectionConnectorSettings = ({ connection: c }: FormProps) => {
	const [fields, setFields] = useState<ConnectorFieldInterface[]>([]);
	const [actions, setActions] = useState<ConnectorAction[]>([]);
	const [values, setValues] = useState<UIValues>({});

	const { api, handleError, showStatus, redirect } = useContext(AppContext);

	const confirmationButtonsRef = useRef<FeedbackButtonRef[]>([]);

	useEffect(() => {
		const fetchUI = async () => {
			let ui: UIResponse;
			try {
				ui = await api.workspaces.connections.ui(c.id);
			} catch (err) {
				if (err instanceof NotFoundError) {
					redirect('connections');
					showStatus(statuses.connectionDoesNotExistAnymore);
					return;
				}
				if (err instanceof UnprocessableError) {
					if (err.code === 'EventNotExist') {
						console.error(
							`Unprocessable: connection does not implement the 'load' event in its ServeUI method`,
						);
						handleError(
							'An unexpected error has occurred. Please contact the administrator for more information.',
						);
					}
					return;
				}
				handleError(err);
				return;
			}
			setFields(ui.Form.Fields);
			setActions(ui.Form.Actions);
			setValues(ui.Form.Values);
		};
		fetchUI();
	}, []);

	const onActionClick = async (eventName: string, confirmationButtonIndex?: number) => {
		let confirmationButton: FeedbackButtonRef | null = null;
		if (confirmationButtonIndex != null) {
			confirmationButton = confirmationButtonsRef.current[confirmationButtonIndex];
		}
		const hasConfirmationButton = confirmationButton != null;

		// remove the errors
		const fls: ConnectorFieldInterface[] = [];
		for (const f of fields) {
			if ('Error' in f) {
				if (f.Error) {
					f.Error = '';
				}
			}
			fls.push(f);
		}
		setFields(fls);
		if (hasConfirmationButton) {
			confirmationButton!.load();
		}
		let ui: UIResponse;
		try {
			ui = await api.workspaces.connections.uiEvent(c.id, eventName, values);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'EventNotExist') {
					console.error(
						`Unprocessable: connection does not implement the ${eventName} event in its ServeUI method`,
					);
					handleError(
						'An unexpected error has occurred. Please contact the administrator for more information',
					);
				}
				return;
			}
			handleError(err);
			if (hasConfirmationButton) {
				confirmationButton!.stop();
			}
			return;
		}
		if (hasConfirmationButton) {
			confirmationButton!.stop();
		}
		if (eventName === 'save') {
			showStatus(statuses.connectionSaved);
			return;
		}
		if (ui == null) {
			if (hasConfirmationButton) {
				confirmationButton!.confirm();
			}
			return;
		}
		if (ui.Alert != null) {
			showStatus({ variant: ui.Alert.Variant, icon: icons.EXCLAMATION, text: ui.Alert.Message });
		}
		if (ui.Form != null) {
			setFields(ui.Form.Fields);
			setActions(ui.Form.Actions);
			setValues(ui.Form.Values);
		}
	};

	const onFieldChange = (name: string, value: any) => {
		setValues((prevValues) => ({ ...prevValues, [name]: value }));
	};

	const fieldsToRender: ReactNode[] = [];
	for (const [i, f] of fields.entries()) {
		fieldsToRender.push(<ConnectorField key={i} field={f} />);
	}

	const actionsToRender: ReactNode[] = [];
	for (const [i, a] of actions.entries()) {
		if (a.Confirm) {
			actionsToRender.push(
				<FeedbackButton
					key={a.Event}
					variant={a.Variant}
					onClick={async () => {
						await onActionClick(a.Event, i);
					}}
					ref={(ref) => {
						confirmationButtonsRef.current[i] = ref!;
					}}
				>
					{a.Text}
				</FeedbackButton>,
			);
		} else {
			actionsToRender.push(
				<SlButton
					key={a.Event}
					variant={a.Variant}
					onClick={async () => {
						await onActionClick(a.Event);
					}}
				>
					{a.Text}
				</SlButton>,
			);
		}
	}

	return <SettingsForm fields={fieldsToRender} actions={actionsToRender} values={values} onChange={onFieldChange} />;
};

export default ConnectionConnectorSettings;
