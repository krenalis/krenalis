import React, { useState, useEffect, useContext, useRef, ReactNode } from 'react';
import ConnectorField from '../../shared/ConnectorFields/ConnectorField';
import FeedbackButton, { FeedbackButtonRef } from '../../shared/FeedbackButton/FeedbackButton';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import AppContext from '../../../context/AppContext';
import statuses from '../../../constants/statuses';
import * as icons from '../../../constants/icons';
import ConnectorUI from '../../shared/ConnectorUI/ConnectorUI';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import { ConnectorUIResponse, ConnectorValues } from '../../../types/external/api';
import ConnectorFieldInterface, { ConnectorButton } from '../../../types/external/ui';
import { validateConnectorSettings } from '../../../lib/helpers/validateConnectorSettings';

interface FormProps {
	connection: TransformedConnection;
}

const ConnectionConnectorSettings = ({ connection: c }: FormProps) => {
	const [fields, setFields] = useState<ConnectorFieldInterface[]>([]);
	const [buttons, setButtons] = useState<ConnectorButton[]>([]);
	const [values, setValues] = useState<ConnectorValues>({});

	const { api, handleError, showStatus, redirect } = useContext(AppContext);

	const confirmationButtonsRef = useRef<FeedbackButtonRef[]>([]);

	useEffect(() => {
		const fetchUI = async () => {
			let ui: ConnectorUIResponse;
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
						return;
					}
				}
				handleError(err);
				return;
			}
			setFields(ui.Fields);
			setButtons(ui.Buttons);
			setValues(ui.Values);
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
		try {
			validateConnectorSettings(values, fields);
		} catch (err) {
			handleError(err);
			if (hasConfirmationButton) {
				confirmationButton!.stop();
			}
			return;
		}
		let ui: ConnectorUIResponse;
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
					return;
				}
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
		if (ui.Fields != null) {
			setFields(ui.Fields);
			setButtons(ui.Buttons);
			setValues(ui.Values);
		}
	};

	const onFieldChange = (name: string, value: any) => {
		setValues((prevValues) => ({ ...prevValues, [name]: value }));
	};

	const fieldsToRender: ReactNode[] = [];
	for (const [i, f] of fields.entries()) {
		fieldsToRender.push(<ConnectorField key={i} field={f} />);
	}

	let hasSaveButton = false;
	const buttonsToRender: ReactNode[] = [];
	if (buttons) {
		for (const [i, b] of buttons.entries()) {
			if (b.Event !== 'save') {
				buttonsToRender.push(
					<FeedbackButton
						key={b.Event}
						variant={b.Variant}
						onClick={async () => {
							await onActionClick(b.Event, i);
						}}
						ref={(ref) => {
							confirmationButtonsRef.current[i] = ref!;
						}}
					>
						{b.Text}
					</FeedbackButton>,
				);
			} else {
				hasSaveButton = true;
				buttonsToRender.push(
					<SlButton
						key={b.Event}
						variant={b.Variant}
						onClick={async () => {
							await onActionClick(b.Event);
						}}
					>
						{b.Text}
					</SlButton>,
				);
			}
		}
	}

	if (!hasSaveButton) {
		buttonsToRender.push(
			<SlButton variant='primary' onClick={() => onActionClick('save')}>
				Save
			</SlButton>,
		);
	}

	return <ConnectorUI fields={fieldsToRender} buttons={buttonsToRender} values={values} onChange={onFieldChange} />;
};

export default ConnectionConnectorSettings;
