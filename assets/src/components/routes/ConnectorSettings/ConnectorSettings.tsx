import React, { useState, useEffect, useContext, useRef, ReactNode } from 'react';
import './ConnectorSettings.css';
import ConnectorField from '../../base/ConnectorFields/ConnectorField';
import FeedbackButton, { FeedbackButtonRef } from '../../base/FeedbackButton/FeedbackButton';
import NotFound from '../NotFound/NotFound';
import Flex from '../../base/Flex/Flex';
import ConnectorUI from '../../base/ConnectorUI/ConnectorUI';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import TransformedConnector from '../../../lib/core/connector';
import {
	ConnectionRole,
	ConnectionToAdd,
	SendingMode as SendingModeType,
	Strategy,
} from '../../../lib/api/types/connection';
import { ConnectorUIResponse, ConnectorSettings } from '../../../lib/api/types/responses';
import ConnectorFieldInterface, { ConnectorButton } from '../../../lib/api/types/ui';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import { validateConnectorSettings } from '../../../lib/core/connectorSettings';
import * as icons from '../../../constants/icons';

const strategyOptions: Strategy[] = ['Conversion', 'Fusion', 'Isolation', 'Preservation'];

const hasStrategy = (connectionRole: ConnectionRole, c: TransformedConnector): boolean => {
	return connectionRole === 'Source' && c.strategies;
};

const ConnectorSettings = () => {
	const [connector, setConnector] = useState<TransformedConnector | null>(null);
	const [name, setName] = useState<string>('');
	const [strategy, setStrategy] = useState<Strategy | null>(null);
	const [sendingMode, setSendingMode] = useState<SendingModeType | null>(null);
	const [fields, setFields] = useState<ConnectorFieldInterface[]>([]);
	const [buttons, setButtons] = useState<ConnectorButton[]>([]);
	const [settings, setSettings] = useState<ConnectorSettings | null>(null);
	const [newConnectionID, setNewConnectionID] = useState<number>(0);
	const [notFound, setNotFound] = useState<boolean>(false);

	const { api, handleError, showStatus, redirect, connectors, setIsLoadingConnections, setTitle, selectedWorkspace } =
		useContext(AppContext);

	const confirmationButtonsRef = useRef<FeedbackButtonRef[]>([]);

	let connectorName: string, connectionRole: ConnectionRole, authToken: string;
	const url = new URL(document.location.href);
	connectorName = decodeURIComponent(url.pathname.split('/').pop());
	const roleParam = url.searchParams.get('role') as ConnectionRole | null | '';
	if (roleParam == null || roleParam === '') {
		connectionRole = 'Source';
	} else {
		connectionRole = roleParam;
	}
	const token = url.searchParams.get('authToken');
	if (token == null) {
		authToken = '';
	} else {
		authToken = token;
	}

	useEffect(() => {
		const connector = connectors.find((c) => c.name === connectorName);
		if (connector.isFile) {
			redirect(`connectors/file/${encodeURIComponent(connector.name)}?role=${connectionRole}`);
		}
	}, []);

	useEffect(() => {
		if (newConnectionID > 0) {
			redirect(`connections/${newConnectionID}/actions?new=true`);
		}
	}, [newConnectionID]);

	useEffect(() => {
		const fetchData = async () => {
			const connector = connectors.find((c) => c.name === connectorName);
			if (connector == null) {
				setNotFound(true);
				return;
			}
			setConnector(connector);
			setTitle(
				<Flex alignItems='baseline' gap={10}>
					<span style={{ position: 'relative', top: '3px' }}>{getConnectorLogo(connector.icon)}</span>
					<span>
						Add {connector.name} {connectionRole.toLowerCase()} connection
					</span>
				</Flex>,
			);
			setName(connector.name);
			if (hasStrategy(connectionRole, connector)) {
				setStrategy(strategyOptions[0]);
			}
			const supportedModes = connector.supportedSendingModes;
			if (connectionRole !== 'Source' && supportedModes.length > 0) {
				setSendingMode(supportedModes[0]);
			}
			if (!connector.hasSettings(connectionRole)) return;
			let ui: ConnectorUIResponse;
			try {
				ui = await api.connectors.ui(selectedWorkspace, connectorName, connectionRole, authToken);
			} catch (err) {
				if (err instanceof NotFoundError) {
					redirect('connectors');
					handleError('The connector does not exist anymore');
					return;
				}
				if (err instanceof UnprocessableError) {
					if (err.code === 'EventNotExist') {
						console.error(
							`Unprocessable: connector does not implement the 'load' event in its ServeUI method`,
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
		};
		fetchData();
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
			if ('error' in f) {
				if (f.error) {
					f.error = '';
				}
			}
			fls.push(f);
		}
		setFields(fls);
		if (hasConfirmationButton) {
			confirmationButton!.load();
		}
		if (eventName === 'save') {
			try {
				validateConnectorSettings(settings, fields);
			} catch (err) {
				handleError(err);
				if (hasConfirmationButton) {
					confirmationButton!.stop();
				}
				return;
			}
			let id: number;
			try {
				const connection: ConnectionToAdd = {
					name: name,
					role: connectionRole,
					connector: connectorName,
					strategy: strategy,
					sendingMode: sendingMode,
					settings: settings,
					linkedConnections: null,
				};
				id = await api.workspaces.createConnection(connection, authToken);
			} catch (err) {
				if (err instanceof UnprocessableError) {
					if (err.code === 'ConnectorNotExist') {
						redirect('connectors');
					}
				}
				if (hasConfirmationButton) {
					confirmationButton!.stop();
				}
				handleError(err);
				return;
			}
			if (hasConfirmationButton) {
				confirmationButton!.stop();
			}
			setNewConnectionID(id);
			setIsLoadingConnections(true);
			return;
		}
		let ui: ConnectorUIResponse;
		try {
			ui = await api.connectors.uiEvent(
				selectedWorkspace,
				connectorName,
				eventName,
				settings,
				connectionRole,
				authToken,
			);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				if (err.code === 'EventNotExist') {
					console.error(
						`Unprocessable: connection does not implement the ${eventName} event in its ServeUI method`,
					);
					handleError(
						'An unexpected error has occurred. Please contact the administrator for more information.',
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
	for (const f of fields) {
		fieldsToRender.push(<ConnectorField key={f.label} field={f} />);
	}

	const buttonsToRender: ReactNode[] = [];
	if (buttons) {
		for (const [i, b] of buttons.entries()) {
			buttonsToRender.push(
				<FeedbackButton
					key={b.event}
					variant={b.variant}
					onClick={async () => {
						await onActionClick(b.event, i);
					}}
					ref={(ref) => {
						confirmationButtonsRef.current[i] = ref!;
					}}
				>
					{b.text}
				</FeedbackButton>,
			);
		}
	}
	buttonsToRender.push(
		<div className='connector-settings__save-wrapper'>
			<SlButton
				className='connector-settings__save-button'
				variant='primary'
				onClick={() => onActionClick('save')}
			>
				Add
			</SlButton>
		</div>,
	);

	if (notFound) {
		return <NotFound />;
	}

	const c = connector;
	if (connector == null) {
		return null;
	}

	const showStrategy = hasStrategy(connectionRole, c);

	return (
		<div className='connector-settings'>
			<div className='route-content'>
				<div className='connector-settings__settings'>
					<div className='connector-settings__basic'>
						<div className='connector-settings__input'>
							<SlInput
								className='connector-settings__name-field'
								name='name'
								value={name}
								label='Name'
								type='text'
								onSlInput={(e) => {
									const target = e.currentTarget as any;
									setName(target!.value);
								}}
							/>
						</div>
						{showStrategy && (
							<div className='connector-settings__input'>
								<SlSelect
									className='connector-settings__strategy-field'
									name='strategy'
									value={strategy || 'Conversion'}
									label='Strategy'
									onSlChange={(e) => {
										const target = e.currentTarget as any;
										const value = target.value as Strategy;
										setStrategy(value);
									}}
								>
									{strategyOptions.map((option) => (
										<SlOption value={option}>{option}</SlOption>
									))}
								</SlSelect>
							</div>
						)}
						{connectionRole !== 'Source' && connector.supportedSendingModes.length > 0 && (
							<div className='connector-settings__input'>
								<SlSelect
									className='connector-settings__mode-field'
									name='mode'
									value={sendingMode}
									label='Sending mode'
									onSlChange={(e) => {
										const target = e.currentTarget as any;
										const value = target.value as SendingModeType;
										setSendingMode(value);
									}}
								>
									<div className='connector-settings__mode-value-icon' slot='prefix'>
										<SlIcon
											name={
												sendingMode === 'Server'
													? 'cloud'
													: sendingMode === 'Client'
														? 'phone'
														: 'send'
											}
										/>
									</div>
									{connector.supportedSendingModes.map((m) => (
										<SlOption key={m} value={m}>
											<div slot='prefix'>
												<SlIcon
													className='connector-settings__mode-icon'
													name={m === 'Server' ? 'cloud' : m === 'Client' ? 'phone' : 'send'}
												/>
											</div>
											{m === 'ClientAndServer' ? 'Client and server' : m}
										</SlOption>
									))}
								</SlSelect>
							</div>
						)}
					</div>
					{connector.hasSettings(connectionRole) ? (
						<ConnectorUI
							fields={fieldsToRender}
							buttons={buttonsToRender}
							settings={settings}
							onChange={onFieldChange}
						/>
					) : (
						<>{buttonsToRender}</>
					)}
				</div>
			</div>
		</div>
	);
};

export default ConnectorSettings;
