import React, { useState, useEffect, useContext, useRef, ReactNode } from 'react';
import ConnectorField from '../../base/ConnectorFields/ConnectorField';
import FeedbackButton, { FeedbackButtonRef } from '../../base/FeedbackButton/FeedbackButton';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import AppContext from '../../../context/AppContext';
import * as icons from '../../../constants/icons';
import ConnectorUI from '../../base/ConnectorUI/ConnectorUI';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import TransformedConnection from '../../../lib/core/connection';
import { ConnectorUIResponse, ConnectorSettings } from '../../../lib/api/types/responses';
import ConnectorFieldInterface, { ConnectorButton } from '../../../lib/api/types/ui';
import { validateConnectorSettings } from '../../../lib/core/connectorSettings';

interface FormProps {
	connection: TransformedConnection;
}

const ConnectionConnectorSettings = ({ connection: c }: FormProps) => {
	const [fields, setFields] = useState<ConnectorFieldInterface[]>([]);
	const [buttons, setButtons] = useState<ConnectorButton[]>([]);
	const [settings, setSettings] = useState<ConnectorSettings>({});
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const { api, handleError, showStatus, redirect } = useContext(AppContext);

	const confirmationButtonsRef = useRef<FeedbackButtonRef[]>([]);

	useEffect(() => {
		const fetchUI = async () => {
			setIsLoading(true);
			let ui: ConnectorUIResponse;
			try {
				ui = await api.workspaces.connections.ui(c.id);
			} catch (err) {
				setIsLoading(false);
				if (err instanceof NotFoundError) {
					redirect('connections');
					handleError('The connection does not exist anymore');
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
			setFields(ui.fields);
			setButtons(ui.buttons);
			setSettings(ui.settings);
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		fetchUI();
	}, []);

	const onButtonClick = async (eventName: string, confirmationButtonIndex?: number) => {
		let confirmationButton: FeedbackButtonRef | null = null;
		if (confirmationButtonIndex != null) {
			confirmationButton = confirmationButtonsRef.current[confirmationButtonIndex];
		}
		const hasConfirmationButton = confirmationButton != null;

		// remove the errors
		const fls: ConnectorFieldInterface[] = [];
		for (const f of fields) {
			const fc = structuredClone(f);
			if ('error' in fc) {
				if (fc.error) {
					fc.error = '';
				}
			}
			fls.push(fc);
		}
		setFields(fls);
		if (hasConfirmationButton) {
			confirmationButton!.load();
		}
		try {
			validateConnectorSettings(settings, fields);
		} catch (err) {
			handleError(err);
			if (hasConfirmationButton) {
				confirmationButton!.stop();
			}
			return;
		}
		let ui: ConnectorUIResponse;
		try {
			ui = await api.workspaces.connections.uiEvent(c.id, eventName, settings);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				handleError('The connection does not exist anymore');
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
			showStatus({ variant: 'success', icon: icons.OK, text: 'The connection settings have been saved' });
			return;
		}
		if (ui == null) {
			if (hasConfirmationButton) {
				confirmationButton!.confirm();
			}
			return;
		}
		if (ui.alert != null) {
			showStatus({ variant: ui.alert.variant, icon: icons.EXCLAMATION, text: ui.alert.message });
		}
		if (ui.fields != null) {
			setFields(ui.fields);
			setButtons(ui.buttons);
			setSettings(ui.settings);
		}
	};

	const onFieldChange = (name: string, value: any) => {
		setSettings((prevSettings) => ({ ...prevSettings, [name]: value }));
	};

	const fieldsToRender: ReactNode[] = [];
	for (const [i, f] of fields.entries()) {
		fieldsToRender.push(<ConnectorField key={i} field={f} />);
	}

	const buttonsToRender: ReactNode[] = [];
	for (const [i, b] of buttons.entries()) {
		buttonsToRender.push(
			<FeedbackButton
				key={b.event}
				name={b.event}
				variant={b.event === 'save' ? 'primary' : b.variant}
				onClick={async () => {
					await onButtonClick(b.event, i);
				}}
				ref={(ref) => {
					confirmationButtonsRef.current[i] = ref!;
				}}
			>
				{b.event === 'save' ? 'Save' : b.text}
			</FeedbackButton>,
		);
	}

	if (isLoading) {
		return (
			<SlSpinner
				style={
					{
						fontSize: '2.5rem',
						'--track-width': '5px',
					} as React.CSSProperties
				}
			/>
		);
	}

	return (
		<ConnectorUI fields={fieldsToRender} buttons={buttonsToRender} settings={settings} onChange={onFieldChange} />
	);
};

export default ConnectionConnectorSettings;
