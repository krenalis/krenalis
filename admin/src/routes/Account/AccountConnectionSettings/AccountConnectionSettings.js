import { useState, useEffect, useRef } from 'react';
import './AccountConnectionSettings.css';
import ConnectorField from '../../../components/ConnectorFields/ConnectorField';
import NotFound from '../../NotFound/NotFound';
import Breadcrumbs from '../../../components/Breadcrumbs/Breadcrumbs';
import Toast from '../../../components/Toast/Toast';
import call from '../../../utils/call';
import { SettingsContext } from '../../../context/SettingsContext';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

const AccountConnectionSettings = () => {
	let [connection, setConnection] = useState({});
	let [fields, setFields] = useState([]);
	let [actions, setActions] = useState([]);
	let [values, setValues] = useState(null);
	let [status, setStatus] = useState(null);
	let [notFound, setNotFound] = useState(false);

	const toastRef = useRef();
	const connectionID = Number(String(window.location).split('/').at(-2));

	useEffect(() => {
		const fetchData = async (path, callback) => {
			let [res, err] = await call(path, connectionID);
			if (err !== null) {
				setStatus({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
				toastRef.current.toast();
				return;
			}
			callback(res);
		};

		fetchData('/admin/connections/get', (connection) => {
			if (connection == null) {
				setNotFound(true);
				return;
			}
			setConnection(connection);
		});

		fetchData('/admin/connectors/ui', (ui) => {
			setFields(ui.Form.Fields);
			setActions(ui.Form.Actions);
			setValues(ui.Form.Values);
		});
	}, []);

	const onActionClick = async (e) => {
		// remove the errors
		let fls = [];
		for (let f of fields) {
			f.Error = '';
			fls.push(f);
		}
		setFields(fls);

		let [ui, err] = await call('/admin/connectors/ui-event', {
			connection: connectionID,
			event: e,
			values: values,
		});
		if (err != null) {
			setStatus({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			toastRef.current.toast();
			return;
		}
		if (ui.Alert != null) {
			setStatus({ variant: ui.Alert.Variant, icon: 'exclamation-square', text: ui.Alert.Message });
			toastRef.current.toast();
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

	if (notFound) {
		return <NotFound />;
	}

	let connectionName = connection.Name;

	let connectionLogo;
	if (connection.LogoURL !== '') {
		connectionLogo = <img className='littleLogo' src={connection.LogoURL} alt={`${connectionName}'s logo`} />;
	}

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

	return (
		<div className='AccountConnectionSettings'>
			<Breadcrumbs
				breadcrumbs={[
					{ Name: 'Your connections', Link: '/admin/account/connections' },
					{ Name: `${connectionName}'s settings` },
				]}
			/>
			<div className='content'>
				<Toast reactRef={toastRef} status={status} />
				<div className='title'>
					{connectionLogo}
					<div className='text'>Configure {connectionName}</div>
				</div>
				<div className='form'>
					<SettingsContext.Provider value={{ values: values, onChange: onFieldChange }}>
						<div className='fields'>{fieldsToRender}</div>
					</SettingsContext.Provider>
					<div className='actions'>{actionsToRender}</div>
				</div>
			</div>
		</div>
	);
};

export default AccountConnectionSettings;
