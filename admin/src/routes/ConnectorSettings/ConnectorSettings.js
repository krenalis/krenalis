import { useState, useEffect, useRef, useContext } from 'react';
import './ConnectorSettings.css';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import ConnectorField from '../../components/ConnectorFields/ConnectorField';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import NotFound from '../NotFound/NotFound';
import { SettingsContext } from '../../context/SettingsContext';
import { AppContext } from '../../context/AppContext';
import statuses from '../../constants/statuses';
import { NavLink, Navigate } from 'react-router-dom';
import {
	SlButton,
	SlInput,
	SlSelect,
	SlSwitch,
	SlMenuItem,
	SlIcon,
} from '@shoelace-style/shoelace/dist/react/index.js';
import { NotFoundError, UnprocessableError } from '../../api/errors';

const ConnectorSettings = () => {
	let [connector, setConnector] = useState(null);
	let [name, setName] = useState('');
	let [storage, setStorage] = useState(0);
	let [storages, setStorages] = useState([]);
	let [stream, setStream] = useState(0);
	let [streams, setStreams] = useState([]);
	let [websiteHost, setWebsiteHost] = useState('');
	let [isEnabled, setIsEnabled] = useState(true);
	let [fields, setFields] = useState([]);
	let [actions, setActions] = useState([]);
	let [values, setValues] = useState(null);
	let [newConnectionID, setNewConnectionID] = useState(0);
	let [notFound, setNotFound] = useState(false);

	let { API, showError, showStatus, redirect } = useContext(AppContext);

	let connectorID, connectionRole, OAuthToken;
	let url = new URL(document.location);
	connectorID = Number(url.pathname.split('/').pop());
	let roleParam = url.searchParams.get('role');
	if (roleParam == null || roleParam === '') {
		connectionRole = 'Source';
	} else {
		connectionRole = roleParam;
	}
	OAuthToken = url.searchParams.get('oauthToken') == null ? '' : url.searchParams.get('oauthToken');

	useEffect(() => {
		const fetchData = async () => {
			let err, connector;
			[connector, err] = await API.connectors.get(connectorID);
			if (err) {
				if (err instanceof NotFoundError) {
					redirect('/admin/connectors');
					showStatus(statuses.connectorDoesNotExistAnymore);
					return;
				}
				showError(err);
				return;
			}
			if (connector == null) {
				setNotFound(true);
				return;
			}
			setConnector(connector);
			setName(connector.Name);
			let storages = [];
			if (connector.Type === 'File') {
				let connections;
				[connections, err] = await API.connections.find();
				if (err) {
					showError(err);
					return;
				}
				for (let c of connections) {
					if (c.Type === 'Storage' && c.Role === connectionRole) storages.push(c);
				}
			}
			setStorages(storages);
			let streams = [];
			if (
				(connector.Type === 'Mobile' || connector.Type === 'Website' || connector.Type === 'Server') &&
				connectionRole === 'Source'
			) {
				let connections;
				[connections, err] = await API.connections.find();
				if (err) {
					showError(err);
					return;
				}
				for (let c of connections) {
					if (c.Type === 'Stream' && c.Role === connectionRole) streams.push(c);
				}
			}
			setStreams(streams);
			if (connector.HasSettings === false) return;
			let ui;
			[ui, err] = await API.connectors.ui(connectorID, connectionRole, OAuthToken);
			if (err) {
				if (err instanceof NotFoundError) {
					redirect('/admin/connectors');
					showStatus(statuses.connectorDoesNotExistAnymore);
					return;
				}
				if (err instanceof UnprocessableError) {
					if (err.code === 'EventNotExist') {
						// TODO(@Andrea): find a way to show the full error message
						// in the toast notification when the server is started with
						// the CHICHI_DEBUG_UI environment variable set to 'true'.
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

	const onActionClick = async (e) => {
		// remove the errors
		let fls = [];
		for (let f of fields) {
			f.Error = '';
			fls.push(f);
		}
		setFields(fls);
		if (e === 'save') {
			let [id, err] = await API.workspace.addConnection(connectorID, connectionRole, values, {
				Name: name,
				Enabled: isEnabled,
				Storage: storage,
				Stream: stream,
				WebsiteHost: websiteHost,
				OAuth: OAuthToken,
			});
			if (err != null) {
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'ConnectorNotExist':
							redirect('/admin/connectors');
							showStatus(statuses.connectorDoesNotExistAnymore);
							break;
						case 'InvalidSettings':
							showStatus(statuses.settingsNotValid);
							break;
						case 'StorageNotExist':
							showStatus(statuses.storageNotExist);
							break;
						case 'StreamNotExist':
							showStatus(statuses.streamNotExist);
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
			return;
		}
		let [ui, err] = await API.connectors.uiEvent(connectorID, e, values, connectionRole, OAuthToken);
		if (err != null) {
			if (err instanceof UnprocessableError) {
				if (err.code === 'EventNotExist') {
					// TODO(@Andrea): find a way to show the full error message
					// in the toast notification when the server is started with
					// the CHICHI_DEBUG_UI environment variable set to 'true'.
					console.error(`Unprocessable: connection does not implement the ${e} event in its ServeUI method`);
					showError('Unexpected error. Contact the administrator for more informations.');
				}
				return;
			}
			showError(err);
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

	const onSave = async () => {
		let [id, err] = await API.workspace.addConnection(connectorID, connectionRole, values, {
			Name: name,
			Enabled: isEnabled,
			Storage: storage,
			Stream: stream,
			WebsiteHost: websiteHost,
			OAuth: OAuthToken,
		});
		if (err != null) {
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'ConnectorNotExist':
						redirect('/admin/connectors');
						showStatus(statuses.connectorDoesNotExistAnymore);
						break;
					case 'InvalidSettings':
						showStatus(statuses.settingsNotValid);
						break;
					case 'StorageNotExist':
						showStatus(statuses.storageNotExist);
						break;
					case 'StreamNotExist':
						showStatus(statuses.streamNotExist);
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
		return;
	};

	let fieldsToRender = [];
	for (let f of fields) {
		fieldsToRender.push(<ConnectorField field={f} />);
	}

	let actionsToRender = [];
	for (let a of actions) {
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

	if (notFound) {
		return <NotFound />;
	}

	if (newConnectionID > 0) {
		return <Navigate to={`/admin/connections?new=${newConnectionID}`} />;
	}

	let c = connector;
	if (connector == null) {
		return;
	}

	return (
		<div className='ConnectorSettings'>
			<PrimaryBackground height={250} overlap={50}>
				<Breadcrumbs
					breadcrumbs={[
						{ Name: 'Connections', Link: '/admin/connections' },
						{ Name: `Add a new ${connectionRole}`, Link: `/admin/connectors/?role=${connectionRole}` },
						{ Name: `Add a ${connector.Name} connection` },
					]}
					onAccent={true}
				/>
				<div className='heading'>
					{c.LogoURL !== '' && <img className='littleLogo' src={c.LogoURL} alt={`${c.Name}'s logo`} />}
					<div className='text'>Add a {c.Name} connection</div>
				</div>
			</PrimaryBackground>
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
						{c.Type === 'File' && (
							<div className='inputWrapper'>
								{storages.length === 0 ? (
									<div className='noStorages'>
										<div className='text'>
											Currently there are no storage {connectionRole.toLowerCase()}s available
										</div>
										<SlButton variant='neutral'>
											<SlIcon name='plus-circle' slot='suffix'></SlIcon>
											Create one
											<NavLink to={`/admin/connectors?role=${connectionRole}`}></NavLink>
										</SlButton>
									</div>
								) : (
									<SlSelect
										name='storage'
										value={storage}
										label='Storage'
										onSlChange={(e) => setStorage(e.currentTarget.value)}
									>
										{storages.map((s) => {
											return <SlMenuItem value={s.ID}>{s.Name}</SlMenuItem>;
										})}
									</SlSelect>
								)}
							</div>
						)}
						{(c.Type === 'Mobile' || c.Type === 'Website' || c.Type === 'Server') &&
							connectionRole === 'Source' && (
								<div className='inputWrapper'>
									{streams.length === 0 ? (
										<div className='noStreams'>
											<div className='text'>
												Currently there are no stream {connectionRole.toLowerCase()}s available
											</div>
											<SlButton variant='neutral'>
												<SlIcon name='plus-circle' slot='suffix'></SlIcon>
												Create one
												<NavLink to={`/admin/connectors?role=${connectionRole}`}></NavLink>
											</SlButton>
										</div>
									) : (
										<SlSelect
											name='stream'
											value={stream}
											label='Stream'
											onSlChange={(e) => setStream(e.currentTarget.value)}
										>
											{streams.map((s) => {
												return <SlMenuItem value={s.ID}>{s.Name}</SlMenuItem>;
											})}
										</SlSelect>
									)}
								</div>
							)}
						{c.Type === 'Website' && (
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
						<div className='switchWrapper'>
							<SlSwitch checked={isEnabled} onSlChange={(e) => setIsEnabled(!isEnabled)}>
								Enable the connection after creation
							</SlSwitch>
						</div>
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
