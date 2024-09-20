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
import { ConnectorUIResponse, ConnectorValues } from '../../../lib/api/types/responses';
import ConnectorFieldInterface, { ConnectorButton } from '../../../lib/api/types/ui';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import { validateConnectorSettings } from '../../../lib/core/connectorSettings';
import { isEventConnection } from '../../../lib/core/connection';
import { LinkedConnectionSelector } from '../../base/LinkedConnectionSelector/LinkedConnectionSelector';
import * as icons from '../../../constants/icons';

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
	const [linkedConnections, setLinkedConnections] = useState<Number[]>();
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
					linkedConnections: linkedConnections,
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
	for (const f of fields) {
		fieldsToRender.push(<ConnectorField key={f.Label} field={f} />);
	}

	const buttonsToRender: ReactNode[] = [];
	if (buttons) {
		for (const [i, b] of buttons.entries()) {
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

	let linkedConnectionsContainer: ReactNode = null;
	if (isEventConnection(connectionRole, connector.type, connector.targets)) {
		linkedConnectionsContainer = (
			<div className='connector-settings__linked-connections'>
				<LinkedConnectionSelector
					linkedConnections={linkedConnections}
					setLinkedConnections={setLinkedConnections}
					connections={connections}
					role={connectionRole}
					title={
						<div className='connector-settings__linked-connections-label'>
							Linked {connectionRole === 'Source' ? 'destinations' : 'sources'}
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
								onSlChange={(e) => {
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
									value={strategy || 'AB-C'}
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
									value={SendingMode}
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
													className='connector-settings__mode-icon'
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
								<div className='connector-settings__input'>
									<SlInput
										className='connector-settings__host-field'
										name='host'
										value={websiteHost}
										placeholder='www.example.com:443'
										label='Host'
										type='text'
										onSlChange={(e) => {
											const target = e.currentTarget as any;
											setWebsiteHost(target.value);
										}}
									/>
								</div>
							</>
						)}
					</div>
					{fieldsToRender.length > 0 ? (
						<ConnectorUI
							fields={fieldsToRender}
							buttons={buttonsToRender}
							values={values}
							onChange={onFieldChange}
						>
							{linkedConnectionsContainer}
						</ConnectorUI>
					) : (
						<>
							{linkedConnectionsContainer}
							{buttonsToRender}
						</>
					)}
				</div>
			</div>
		</div>
	);
};

export default ConnectorSettings;
