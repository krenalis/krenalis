import { useState, useEffect, useContext, useRef } from 'react';
import './ConnectorSettings.css';
import ConnectorField from '../../shared/ConnectorFields/ConnectorField';
import ConfirmationButton from '../../shared/ConfirmationButton/ConfirmationButton';
import NotFound from '../NotFound/NotFound';
import Flex from '../../shared/Flex/Flex';
import { SettingsContext } from '../../../context/SettingsContext';
import { AppContext } from '../../../context/providers/AppProvider';
import statuses from '../../../constants/statuses';
import { SlButton, SlInput, SlSelect, SlOption } from '@shoelace-style/shoelace/dist/react/index.js';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';

const ConnectorSettings = () => {
	const [connector, setConnector] = useState(null);
	const [name, setName] = useState('');
	const [storage, setStorage] = useState(0);
	const [storages, setStorages] = useState([]);
	const [compression, setCompression] = useState('');
	const [websiteHost, setWebsiteHost] = useState('');
	const [fields, setFields] = useState([]);
	const [actions, setActions] = useState([]);
	const [values, setValues] = useState(null);
	const [newConnectionID, setNewConnectionID] = useState(0);
	const [notFound, setNotFound] = useState(false);

	const { api, showError, showStatus, redirect, connectors, connections, setAreConnectionsStale, setTitle } =
		useContext(AppContext);

	const confirmationButtonsRef = useRef([]);

	let connectorID, connectionRole, OAuthToken;
	const url = new URL(document.location);
	connectorID = Number(url.pathname.split('/').pop());
	const roleParam = url.searchParams.get('role');
	if (roleParam == null || roleParam === '') {
		connectionRole = 'Source';
	} else {
		connectionRole = roleParam;
	}
	OAuthToken = url.searchParams.get('oauthToken') == null ? '' : url.searchParams.get('oauthToken');

	useEffect(() => {
		const fetchData = async () => {
			const connector = connectors.find((c) => c.id === connectorID);
			if (connector == null) {
				setNotFound(true);
				return;
			}
			setConnector(connector);
			setTitle(
				<Flex alignItems='baseline' gap='10px'>
					<span style={{ position: 'relative', top: '3px' }}>{connector.logo}</span>
					<span>
						Add {connector.name} {connectionRole.toLowerCase()} connection
					</span>
				</Flex>
			);
			setName(connector.name);
			const storages = [];
			if (connector.type === 'File') {
				for (const c of connections) {
					if (c.type === 'Storage' && c.role === connectionRole) storages.push(c);
				}
			}
			setStorages(storages);
			if (connector.hasSettings === false) return;
			const [ui, err] = await api.connectors.ui(connectorID, connectionRole, OAuthToken);
			if (err) {
				if (err instanceof NotFoundError) {
					redirect('connectors');
					showStatus(statuses.connectorDoesNotExistAnymore);
					return;
				}
				if (err instanceof UnprocessableError) {
					if (err.code === 'EventNotExists') {
						console.error(
							`Unprocessable: connector does not implement the 'load' event in its ServeUI method`
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
		fetchData();
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
		if (eventName === 'save') {
			const [id, err] = await api.workspace.addConnection(connectorID, connectionRole, values, {
				Name: name,
				Enabled: true,
				Storage: storage,
				Compression: compression,
				WebsiteHost: websiteHost,
				OAuth: OAuthToken,
			});
			if (confirmationButton != null) {
				confirmationButton.stop();
			}
			if (err != null) {
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'ConnectorNotExists':
							redirect('connectors');
							showStatus(statuses.connectorDoesNotExistAnymore);
							break;
						case 'InvalidSettings':
							showStatus(statuses.settingsNotValid);
							break;
						case 'StorageNotExists':
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
		}
		const [ui, err] = await api.connectors.uiEvent(connectorID, eventName, values, connectionRole, OAuthToken);
		if (confirmationButton != null) {
			confirmationButton.stop();
		}
		if (err != null) {
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
		if (ui == null) {
			if (confirmationButton != null) {
				confirmationButton.confirm();
			}
			return;
		}
		if (ui.Alert != null) {
			showStatus([ui.Alert.Variant, 'exclamation-square', ui.Alert.Message]);
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

	const onCreateNewStorageClick = () => {
		redirect(`connectors?role=${connectionRole}`);
	};

	const onSave = async () => {
		const [id, err] = await api.workspace.addConnection(connectorID, connectionRole, values, {
			Name: name,
			Enabled: true,
			Storage: storage,
			WebsiteHost: websiteHost,
			OAuth: OAuthToken,
		});
		if (err != null) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'ConnectorNotExists':
						redirect('connectors');
						showStatus(statuses.connectorDoesNotExistAnymore);
						break;
					case 'InvalidSettings':
						showStatus(statuses.settingsNotValid);
						break;
					case 'StorageNotExists':
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

	if (notFound) {
		return <NotFound />;
	}

	if (newConnectionID > 0) {
		redirect(`connections?newConnection=${newConnectionID}`);
	}

	const c = connector;
	if (connector == null) {
		return;
	}

	return (
		<div className='connectorSettings'>
			<div className='routeContent'>
				<div className='settings'>
					<div className='basic'>
						<div className='inputWrapper'>
							<SlInput
								name='name'
								value={name}
								label='Name'
								type='text'
								onSlChange={(e) => setName(e.currentTarget.value)}
							/>
						</div>
						{c.type === 'File' && (
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
										name='storage'
										value={String(storage)}
										label='Storage'
										onSlChange={(e) => setStorage(Number(e.currentTarget.value))}
									>
										{storages.map((s) => {
											return <SlOption value={s.id}>{s.name}</SlOption>;
										})}
									</SlSelect>
								)}
							</div>
						)}
						{c.type === 'File' && (
							<div className='inputWrapper'>
								<SlSelect
									name='compression'
									value={compression}
									label='Compression'
									disabled={storage === 0}
									onSlChange={(e) => setCompression(e.currentTarget.value)}
								>
									<SlOption value=''>None</SlOption>
									<SlOption value='Zip'>Zip</SlOption>
									<SlOption value='Gzip'>Gzip</SlOption>
									<SlOption value='Snappy'>Snappy</SlOption>
								</SlSelect>
							</div>
						)}
						{c.type === 'Website' && (
							<>
								<div className='inputWrapper'>
									<SlInput
										name='host'
										value={websiteHost}
										placeholder='www.example.com:443'
										label='Host'
										type='text'
										onSlChange={(e) => setWebsiteHost(e.currentTarget.value)}
									/>
								</div>
							</>
						)}
					</div>
					{(fieldsToRender.length > 0 || actionsToRender.length > 0) && (
						<div className='form'>
							<SettingsContext.Provider value={{ values: values, onChange: onFieldChange }}>
								<div className='fields'>{fieldsToRender}</div>
							</SettingsContext.Provider>
							<div className='actions'>{actionsToRender}</div>
						</div>
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
