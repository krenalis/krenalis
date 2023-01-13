import { useState, useEffect, useRef } from 'react';
import './ConnectionSQL.css';
import NotFound from '../NotFound/NotFound';
import Toast from '../../components/Toast/Toast';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import NavigationTabs from '../../components/NavigationTabs/NavigationTabs';
import ConnectionHeading from '../../components/ConnectionHeading/ConnectionHeading';
import Grid from '../../components/Grid/Grid';
import call from '../../utils/call';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';
import Editor from '@monaco-editor/react';

const queryMaxSize = 16777215;

const ConnectionSQL = () => {
	let [connection, setConnection] = useState({});
	let [status, setStatus] = useState(null);
	let [notFound, setNotFound] = useState(false);
	let [query, setQuery] = useState('');
	let [limit, setLimit] = useState(20); // TODO(@Andrea): implement as a select
	let [table, setTable] = useState(null);

	let toastRef = useRef();
	let connectionID = Number(String(window.location).split('/').at(-2));

	const onError = (err) => {
		setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
		toastRef.current.toast();
		return;
	};

	useEffect(() => {
		const fetchConnection = async () => {
			let [connection, err] = await call('/admin/connections/get', 'POST', connectionID);
			if (err) {
				onError(err);
				return;
			}
			if (connection == null) {
				setNotFound(true);
				return;
			}
			setConnection(connection);
			setQuery(connection.UsersQuery);
		};
		fetchConnection();
	}, []);

	const handlePreview = async () => {
		if (query.length > queryMaxSize) {
			onError('You query is too long');
			return;
		}
		if (!query.includes(':limit')) {
			onError(`Your query does not contain the ':limit' placeholder`);
			return;
		}
		let [table, err] = await call('/admin/connections/preview-query', 'POST', {
			Connection: connection.ID,
			Query: query,
			Limit: limit,
		});
		if (err !== null) {
			onError(err);
			return;
		}
		if (table.Columns.length === 0) {
			onError('Your query did not return any columns');
			return;
		}
		if (table.Rows.length === 0) {
			onError('Your query did not return any rows');
			return;
		}
		setTable(table);
	};

	const saveQuery = async () => {
		if (query.length > queryMaxSize) {
			onError('You query is too long');
			return;
		}
		if (!query.includes(':limit')) {
			onError(`Your query does not contain the ':limit' placeholder`);
			return;
		}
		let [, err] = await call('/admin/connections/set-users-query', 'POST', {
			Connection: connection.ID,
			Query: query,
		});
		if (err !== null) {
			onError(err);
			return;
		}
		setStatus({ variant: 'success', icon: 'check2-circle', text: 'Your query has been successfully saved' });
		toastRef.current.toast();
	};

	if (notFound) {
		return <NotFound />;
	}

	let c = connection;
	let tabs = [
		{ Name: 'Overview', Link: `/admin/connections/${c.ID}`, Selected: false },
		{ Name: 'SQL query', Link: `/admin/connections/${c.ID}/sql`, Selected: true },
		{ Name: 'Settings', Link: `/admin/connections/${c.ID}/settings`, Selected: false },
	];
	if (c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') {
		tabs.splice(2, 0, { Name: 'Properties', Link: `/admin/connections/${c.ID}/properties`, Selected: false });
	}
	return (
		<div className='ConnectionSQL'>
			<PrimaryBackground contentWidth={1400} height={300}>
				<Breadcrumbs
					onAccent={true}
					breadcrumbs={[{ Name: 'Connections', Link: '/admin/connections' }, { Name: `${c.Name}` }]}
				/>
				<ConnectionHeading connection={c} />
				<NavigationTabs tabs={tabs} onAccent={true} />
			</PrimaryBackground>
			<div className='routeContent'>
				<Toast reactRef={toastRef} status={status} />
				<div className='editorWrapper'>
					<Editor
						onChange={(value) => setQuery(value)}
						defaultLanguage='sql'
						value={query}
						theme='vs-primary'
					/>
				</div>
				<div className='buttons'>
					<SlButton className='previewButton' variant='neutral' size='large' onClick={handlePreview}>
						<SlIcon slot='prefix' name='eye' />
						Preview
					</SlButton>
					<SlButton className='saveButton' variant='primary' size='large' onClick={saveQuery}>
						<SlIcon slot='prefix' name='save' />
						Save
					</SlButton>
				</div>
			</div>
			{table && (
				<SlDialog
					label='Users preview'
					open={true}
					style={{ '--width': '1200px' }}
					onSlAfterHide={() => setTable(null)}
				>
					<Grid table={table} />
				</SlDialog>
			)}
		</div>
	);
};

export default ConnectionSQL;
