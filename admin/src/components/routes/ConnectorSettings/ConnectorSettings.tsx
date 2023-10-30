import React, { useState, useEffect, useContext, useRef, ReactNode } from 'react';
import './ConnectorSettings.css';
import ConnectorField from '../../shared/ConnectorFields/ConnectorField';
import ConfirmationButton, { ConfirmationButtonRef } from '../../shared/ConfirmationButton/ConfirmationButton';
import NotFound from '../NotFound/NotFound';
import Flex from '../../shared/Flex/Flex';
import SettingsForm from '../../shared/SettingsForm/SettingsForm';
import { AppContext } from '../../../context/providers/AppProvider';
import statuses from '../../../constants/statuses';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import TransformedConnector from '../../../lib/helpers/transformedConnector';
import { Compression, ConnectionRole, ConnectionToAdd } from '../../../types/external/connection';
import { UIResponse, UIValues } from '../../../types/external/api';
import ConnectorFieldInterface, { ConnectorAction } from '../../../types/external/ui';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import { ShoelaceEventTarget } from '../../../types/internal/app';

const ConnectorSettings = () => {
	const [connector, setConnector] = useState<TransformedConnector | null>(null);
	const [name, setName] = useState<string>('');
	const [storage, setStorage] = useState<number>(0);
	const [storages, setStorages] = useState<TransformedConnection[]>([]);
	const [compression, setCompression] = useState<Compression>('');
	const [websiteHost, setWebsiteHost] = useState<string>('');
	const [fields, setFields] = useState<ConnectorFieldInterface[]>([]);
	const [actions, setActions] = useState<ConnectorAction[]>([]);
	const [values, setValues] = useState<UIValues>({});
	const [newConnectionID, setNewConnectionID] = useState<number>(0);
	const [notFound, setNotFound] = useState<boolean>(false);

	const { api, showError, showStatus, redirect, connectors, connections, setAreConnectionsStale, setTitle } =
		useContext(AppContext);

	const confirmationButtonsRef = useRef<ConfirmationButtonRef[]>([]);

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
			const storages: TransformedConnection[] = [];
			if (connector.type === 'File') {
				for (const c of connections) {
					if (c.type === 'Storage' && c.role === connectionRole) storages.push(c);
				}
			}
			setStorages(storages);
			if (connector.hasSettings === false) return;
			let ui: UIResponse;
			try {
				ui = await api.connectors.ui(connectorID, connectionRole, OAuthToken);
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
						showError(
							'An unexpected error has occurred. Please contact the administrator for more information.',
						);
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
		fetchData();
	}, []);

	const onActionClick = async (eventName: string, confirmationButtonIndex?: number) => {
		let confirmationButton: ConfirmationButtonRef | null = null;
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
			let id: number;
			try {
				const connection: ConnectionToAdd = {
					name: name,
					role: connectionRole,
					enabled: true,
					connector: connectorID,
					storage: storage,
					compression: compression,
					websiteHost: websiteHost,
					settings: values,
				};
				id = await api.workspaces.addConnection(connection, OAuthToken);
			} catch (err) {
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'ConnectorNotExist':
							redirect('connectors');
							showStatus(statuses.connectorDoesNotExistAnymore);
							break;
						case 'InvalidSettings':
							showStatus(statuses.settingsNotValid);
							break;
						case 'StorageNotExist':
							showStatus(statuses.storageNotExist);
							break;
						default:
							break;
					}
					return;
				}
				if (hasConfirmationButton) {
					confirmationButton!.stop();
				}
				showError(err);
				return;
			}
			if (hasConfirmationButton) {
				confirmationButton!.stop();
			}
			setNewConnectionID(id);
			setAreConnectionsStale(true);
			return;
		}
		let ui: UIResponse;
		try {
			ui = await api.connectors.uiEvent(connectorID, eventName, values, connectionRole, OAuthToken);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				if (err.code === 'EventNotExist') {
					console.error(
						`Unprocessable: connection does not implement the ${eventName} event in its ServeUI method`,
					);
					showError(
						'An unexpected error has occurred. Please contact the administrator for more information.',
					);
				}
				return;
			}
			showError(err);
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

	const onCreateNewStorageClick = () => {
		redirect(`connectors?role=${connectionRole}`);
	};

	const onSave = async () => {
		let id: number;
		try {
			const connection: ConnectionToAdd = {
				name: name,
				role: connectionRole,
				enabled: true,
				connector: connectorID,
				storage: storage,
				compression: compression,
				websiteHost: websiteHost,
				settings: values,
			};
			id = await api.workspaces.addConnection(connection, OAuthToken);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'ConnectorNotExist':
						redirect('connectors');
						showStatus(statuses.connectorDoesNotExistAnymore);
						break;
					case 'InvalidSettings':
						showStatus(statuses.settingsNotValid);
						break;
					case 'StorageNotExist':
						showStatus(statuses.storageNotExist);
						break;
					default:
						break;
				}
				return;
			}
			showError(err);
			return;
		}
		setNewConnectionID(id);
		setAreConnectionsStale(true);
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
				<ConfirmationButton
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
				</ConfirmationButton>,
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
						{c!.type === 'File' && (
							<div className='inputWrapper'>
								{storages.length === 0 ? (
									<div className='noStorages'>
										<div className='text'>
											Currently there are no storage {connectionRole.toLowerCase()}s available
										</div>
										<SlButton variant='neutral' onClick={onCreateNewStorageClick}>
											Create one...
										</SlButton>
									</div>
								) : (
									<SlSelect
										className='storage'
										name='storage'
										value={String(storage)}
										label='Storage'
										onSlChange={(e) => {
											const target = e.currentTarget as ShoelaceEventTarget;
											setStorage(Number(target.value));
										}}
									>
										{storages.map((s) => {
											return (
												<SlOption key={s.id} value={String(s.id)}>
													{s.name}
												</SlOption>
											);
										})}
									</SlSelect>
								)}
							</div>
						)}
						{c!.type === 'File' && (
							<div className='inputWrapper'>
								<SlSelect
									className='compression'
									name='compression'
									value={compression}
									label='Compression'
									disabled={storage === 0}
									onSlChange={(e) => {
										const target = e.currentTarget as ShoelaceEventTarget;
										const value = target.value as Compression;
										setCompression(value);
									}}
								>
									<SlOption value=''>None</SlOption>
									<SlOption value='Zip'>Zip</SlOption>
									<SlOption value='Gzip'>Gzip</SlOption>
									<SlOption value='Snappy'>Snappy</SlOption>
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
