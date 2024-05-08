import React, { useState, useEffect, useContext, useRef, ReactNode } from 'react';
import './ConnectorSettings.css';
import ConnectorField from '../../shared/ConnectorFields/ConnectorField';
import FeedbackButton, { FeedbackButtonRef } from '../../shared/FeedbackButton/FeedbackButton';
import NotFound from '../NotFound/NotFound';
import Flex from '../../shared/Flex/Flex';
import ConnectorUI from '../../shared/ConnectorUI/ConnectorUI';
import AppContext from '../../../context/AppContext';
import statuses from '../../../constants/statuses';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import TransformedConnector from '../../../lib/helpers/transformedConnector';
import {
	ConnectionRole,
	ConnectionToAdd,
	SendingMode as SendingModeType,
	Strategy,
} from '../../../types/external/connection';
import { ConnectorUIResponse, ConnectorValues } from '../../../types/external/api';
import ConnectorFieldInterface, { ConnectorButton } from '../../../types/external/ui';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import { ShoelaceEventTarget } from '../../../types/internal/app';
import { validateConnectorSettings } from '../../../lib/helpers/validateConnectorSettings';
import { isEventConnection } from '../../../lib/helpers/transformedConnection';
import { EventConnectionSelector } from '../../shared/EventConnectionSelector/EventConnectionSelector';

const strategyOptions: Strategy[] = ['AB-C', 'ABC', 'A-B-C', 'AC-B'];

const hasStrategy = (connectionRole: ConnectionRole, c: TransformedConnector): boolean => {
	return connectionRole === 'Source' && (c.type === 'Mobile' || c.type === 'Website');
};

const ConnectorSettings = () => {
	const [connector, setConnector] = useState<TransformedConnector | null>(null);
	const [name, setName] = useState<string>('');
	const [strategy, setStrategy] = useState<Strategy | null>(null);
	const [websiteHost, setWebsiteHost] = useState<string>('');
	const [SendingMode, setSendingMode] = useState<SendingModeType | null>(null);
	const [eventConnections, setEventConnections] = useState<Number[]>();
	const [fields, setFields] = useState<ConnectorFieldInterface[]>([]);
	const [buttons, setButtons] = useState<ConnectorButton[]>([]);
	const [values, setValues] = useState<ConnectorValues>({});
	const [newConnectionID, setNewConnectionID] = useState<number>(0);
	const [notFound, setNotFound] = useState<boolean>(false);

	const {
		api,
		handleError,
		showStatus,
		redirect,
		connectors,
		connections,
		setIsLoadingConnections,
		setTitle,
		selectedWorkspace,
	} = useContext(AppContext);

	const confirmationButtonsRef = useRef<FeedbackButtonRef[]>([]);

	let connectorName: string, connectionRole: ConnectionRole, OAuthToken: string;
	const url = new URL(document.location.href);
	connectorName = decodeURIComponent(url.pathname.split('/').pop());
	const roleParam = url.searchParams.get('role') as ConnectionRole | null | '';
	if (roleParam == null || roleParam === '') {
		connectionRole = 'Source';
	} else {
		connectionRole = roleParam;
	}
	const token = url.searchParams.get('oauthToken');
	if (token == null) {
		OAuthToken = '';
	} else {
		OAuthToken = token;
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
			if (connector.hasUI === false) return;
			let ui: ConnectorUIResponse;
			try {
				ui = await api.connectors.ui(selectedWorkspace, connectorName, connectionRole, OAuthToken);
			} catch (err) {
				if (err instanceof NotFoundError) {
					redirect('connectors');
					showStatus(statuses.connectorDoesNotExistAnymore);
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
					}
					return;
				}
				handleError(err);
				return;
			}
			setFields(ui.Fields);
			setButtons(ui.Buttons);
			setValues(ui.Values);
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
		if (eventName === 'save') {
			try {
				validateConnectorSettings(values, fields);
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
					enabled: true,
					connector: connectorName,
					strategy: strategy,
					websiteHost: websiteHost,
					SendingMode: SendingMode,
					uiValues: values,
					eventConnections: eventConnections,
				};
				id = await api.workspaces.addConnection(connection, OAuthToken);
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
				values,
				connectionRole,
				OAuthToken,
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
		if (ui == null) {
			if (hasConfirmationButton) {
				confirmationButton!.confirm();
			}
			return;
		}
		if (ui.Alert != null) {
			showStatus({ variant: ui.Alert.Variant, icon: 'exclamation-square', text: ui.Alert.Message });
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

	const onSave = async () => {
		try {
			validateConnectorSettings(values, fields);
		} catch (err) {
			handleError(err);
			return;
		}
		let id: number;
		try {
			const connection: ConnectionToAdd = {
				name: name,
				role: connectionRole,
				enabled: true,
				connector: connectorName,
				strategy: strategy,
				websiteHost: websiteHost,
				SendingMode: SendingMode,
				uiValues: values,
				eventConnections: eventConnections,
			};
			id = await api.workspaces.addConnection(connection, OAuthToken);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				if (err.code === 'ConnectorNotExist') {
					redirect('connectors');
				}
			}
			handleError(err);
			return;
		}
		setNewConnectionID(id);
		setIsLoadingConnections(true);
		return;
	};

	const fieldsToRender: ReactNode[] = [];
	for (const f of fields) {
		fieldsToRender.push(<ConnectorField key={f.Label} field={f} />);
	}

	const buttonsToRender: ReactNode[] = [];
	for (const [i, b] of buttons.entries()) {
		if (b.Confirm) {
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

	if (notFound) {
		return <NotFound />;
	}

	const c = connector;
	if (connector == null) {
		return null;
	}

	const showStrategy = hasStrategy(connectionRole, c);

	let eventConnectionsContainer: ReactNode = null;
	if (isEventConnection(connectionRole, connector.type, connector.targets)) {
		eventConnectionsContainer = (
			<div className='eventConnections'>
				<EventConnectionSelector
					eventConnections={eventConnections}
					setEventConnections={setEventConnections}
					connections={connections}
					role={connectionRole}
					title={
						<div className='eventConnectionLabel'>
							Event {connectionRole === 'Source' ? 'destinations' : 'sources'}
						</div>
					}
					description={
						connectionRole === 'Source'
							? 'The destinations to which the events are dispatched.'
							: 'The sources whose events are dispatched to the destination.'
					}
				/>
			</div>
		);
	}

	return (
		<div className='connectorSettings'>
			<div className='routeContent'>
				<div className='settings'>
					<div className='basic'>
						<div className='inputWrapper'>
							<SlInput
								className='name'
								name='name'
								value={name}
								label='Name'
								type='text'
								onSlChange={(e) => {
									const target = e.currentTarget as ShoelaceEventTarget;
									setName(target!.value);
								}}
							/>
						</div>
						{showStrategy && (
							<div className='inputWrapper'>
								<SlSelect
									className='strategy'
									name='strategy'
									value={strategy || 'AB-C'}
									label='Strategy'
									onSlChange={(e) => {
										const target = e.currentTarget as ShoelaceEventTarget;
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
							<div className='inputWrapper'>
								<SlSelect
									className='mode'
									name='mode'
									value={SendingMode}
									label='Sending mode'
									onSlChange={(e) => {
										const target = e.currentTarget as ShoelaceEventTarget;
										const value = target.value as SendingModeType;
										setSendingMode(value);
									}}
								>
									<div className='modeValueIcon' slot='prefix'>
										<SlIcon
											name={
												SendingMode === 'Cloud'
													? 'cloud'
													: SendingMode === 'Device'
														? 'phone'
														: 'send'
											}
										/>
									</div>
									{connector.supportedSendingModes.map((m) => (
										<SlOption key={m} value={m}>
											<div slot='prefix'>
												<SlIcon
													className='modeIcon'
													name={m === 'Cloud' ? 'cloud' : m === 'Device' ? 'phone' : 'send'}
												/>
											</div>
											{m}
										</SlOption>
									))}
								</SlSelect>
							</div>
						)}
						{c!.type === 'Website' && (
							<>
								<div className='inputWrapper'>
									<SlInput
										className='host'
										name='host'
										value={websiteHost}
										placeholder='www.example.com:443'
										label='Host'
										type='text'
										onSlChange={(e) => {
											const target = e.currentTarget as ShoelaceEventTarget;
											setWebsiteHost(target.value);
										}}
									/>
								</div>
							</>
						)}
					</div>
					{(fieldsToRender.length > 0 || buttonsToRender.length > 0) && (
						<ConnectorUI
							fields={fieldsToRender}
							buttons={buttonsToRender}
							values={values}
							onChange={onFieldChange}
						>
							{eventConnectionsContainer}
						</ConnectorUI>
					)}
					{fieldsToRender.length === 0 && buttonsToRender.length === 0 && (
						<>
							{eventConnectionsContainer}
							<div className='saveWrapper'>
								<SlButton className='saveButton' variant='primary' onClick={onSave}>
									Save
								</SlButton>
							</div>
						</>
					)}
				</div>
			</div>
		</div>
	);
};

export default ConnectorSettings;
