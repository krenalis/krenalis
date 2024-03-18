import React, { useState, useEffect, useContext, useRef, ReactNode } from 'react';
import './ConnectorSettings.css';
import ConnectorField from '../../shared/ConnectorFields/ConnectorField';
import FeedbackButton, { FeedbackButtonRef } from '../../shared/FeedbackButton/FeedbackButton';
import NotFound from '../NotFound/NotFound';
import Flex from '../../shared/Flex/Flex';
import SettingsForm from '../../shared/SettingsForm/SettingsForm';
import AppContext from '../../../context/AppContext';
import statuses from '../../../constants/statuses';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import TransformedConnector from '../../../lib/helpers/transformedConnector';
import { BusinessID, ConnectionRole, ConnectionToAdd, Strategy } from '../../../types/external/connection';
import { UIResponse, UIValues } from '../../../types/external/api';
import ConnectorFieldInterface, { ConnectorAction } from '../../../types/external/ui';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import { ShoelaceEventTarget } from '../../../types/internal/app';
import { validateConnectorSettings } from '../../../lib/helpers/validateConnectorSettings';

const strategyOptions: Strategy[] = ['AB-C', 'ABC', 'A-B-C', 'AC-B'];

const hasStrategy = (connectionRole: ConnectionRole, c: TransformedConnector): boolean => {
	return connectionRole === 'Source' && (c.type === 'Mobile' || c.type === 'Website');
};

const ConnectorSettings = () => {
	const [connector, setConnector] = useState<TransformedConnector | null>(null);
	const [name, setName] = useState<string>('');
	const [strategy, setStrategy] = useState<Strategy | null>(null);
	const [websiteHost, setWebsiteHost] = useState<string>('');
	const [businessID, setBusinessID] = useState<BusinessID>({ Name: '', Label: '' });
	const [fields, setFields] = useState<ConnectorFieldInterface[]>([]);
	const [actions, setActions] = useState<ConnectorAction[]>([]);
	const [values, setValues] = useState<UIValues>({});
	const [newConnectionID, setNewConnectionID] = useState<number>(0);
	const [notFound, setNotFound] = useState<boolean>(false);

	const { api, handleError, showStatus, redirect, connectors, setIsLoadingConnections, setTitle, selectedWorkspace } =
		useContext(AppContext);

	const confirmationButtonsRef = useRef<FeedbackButtonRef[]>([]);

	let connectorID: number, connectionRole: ConnectionRole, OAuthToken: string;
	const url = new URL(document.location.href);
	connectorID = Number(url.pathname.split('/').pop());
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
		const connector = connectors.find((c) => c.id === connectorID);
		if (connector.isFile) {
			redirect(`connectors/file/${connector.id}?role=${connectionRole}`);
		}
	}, []);

	useEffect(() => {
		if (newConnectionID > 0) {
			redirect(`connections/${newConnectionID}/actions?new=true`);
		}
	}, [newConnectionID]);

	useEffect(() => {
		const fetchData = async () => {
			const connector = connectors.find((c) => c.id === connectorID);
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
			if (connector.hasSettings === false) return;
			let ui: UIResponse;
			try {
				ui = await api.connectors.ui(selectedWorkspace, connectorID, connectionRole, OAuthToken);
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
			setFields(ui.Form.Fields);
			setActions(ui.Form.Actions);
			setValues(ui.Form.Values);
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
					connector: connectorID,
					strategy: strategy,
					websiteHost: websiteHost,
					businessID: businessID,
					settings: values,
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
		let ui: UIResponse;
		try {
			ui = await api.connectors.uiEvent(
				selectedWorkspace,
				connectorID,
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
		if (ui.Form != null) {
			setFields(ui.Form.Fields);
			setActions(ui.Form.Actions);
			setValues(ui.Form.Values);
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
				connector: connectorID,
				strategy: strategy,
				websiteHost: websiteHost,
				businessID: businessID,
				settings: values,
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

	if (notFound) {
		return <NotFound />;
	}

	const c = connector;
	if (connector == null) {
		return null;
	}

	const showStrategy = hasStrategy(connectionRole, c);
	const showBusinessID = connectionRole === 'Source' && c.type !== 'Storage' && c.type !== 'Stream';
	const businessIDKind = ['File', 'Database'].includes(c.type) ? 'column' : 'property';

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
							{showBusinessID && (
								<>
									<SlInput
										className='businessIDName'
										name='businessIDName'
										helpText={`The name of the ${businessIDKind} from which the Business ID is read when importing. Can be left empty to indicate to not import it.`}
										placeholder='Something like "email", "customer_id", etc...'
										value={businessID.Name}
										label='Business ID Name'
										type='text'
										maxlength={1024}
										onSlChange={(e) => {
											const target = e.currentTarget as ShoelaceEventTarget;
											setBusinessID({
												Name: target!.value,
												Label: businessID.Label,
											});
										}}
									/>
									<SlInput
										className='businessIDLabel'
										name='businessIDLabel'
										value={businessID.Label}
										placeholder='Something like "Email", "Customer ID", etc...'
										helpText='A human-readable label for the Business ID. Mandatory when a Business ID name is specified.'
										label='Business ID Label'
										type='text'
										maxlength={16}
										onSlChange={(e) => {
											const target = e.currentTarget as ShoelaceEventTarget;
											setBusinessID({
												Name: businessID.Name,
												Label: target!.value,
											});
										}}
									/>
								</>
							)}
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
					{(fieldsToRender.length > 0 || actionsToRender.length > 0) && (
						<SettingsForm
							fields={fieldsToRender}
							actions={actionsToRender}
							values={values}
							onChange={onFieldChange}
						/>
					)}
					{fieldsToRender.length === 0 && actionsToRender.length === 0 && (
						<div className='saveWrapper'>
							<SlButton className='saveButton' variant='primary' onClick={onSave}>
								Save
							</SlButton>
						</div>
					)}
				</div>
			</div>
		</div>
	);
};

export default ConnectorSettings;
