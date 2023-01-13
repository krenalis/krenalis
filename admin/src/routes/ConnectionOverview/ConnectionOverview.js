import { useState, useEffect, useRef } from 'react';
import './ConnectionOverview.css';
import NotFound from '../NotFound/NotFound';
import Toast from '../../components/Toast/Toast';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import NavigationTabs from '../../components/NavigationTabs/NavigationTabs';
import FlexContainer from '../../components/FlexContainer/FlexContainer';
import ConnectionHeading from '../../components/ConnectionHeading/ConnectionHeading';
import call from '../../utils/call';
import { BarChart, Bar, XAxis, Tooltip, YAxis, CartesianGrid } from 'recharts';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionOverview = () => {
	let [connection, setConnection] = useState({});
	let [userStats, setUserStats] = useState([]);
	let [hasImports, setHasImports] = useState(true);
	let [imports, setImports] = useState([]);
	let [askImportConfirmation, setAskImportConfirmation] = useState(false);
	let [resetCursor, setResetCursor] = useState(false);
	let [notFound, setNotFound] = useState(false);
	let [status, setStatus] = useState(null);

	const toastRef = useRef();
	const connectionID = Number(String(window.location).split('/').pop());

	const onError = (err) => {
		setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
		toastRef.current.toast();
		return;
	};

	useEffect(() => {
		const fetchConnection = async () => {
			let err;

			// get the connection.
			let connection;
			[connection, err] = await call('/admin/connections/get', 'POST', connectionID);
			if (err) {
				onError(err);
				return;
			}
			if (connection == null) {
				setNotFound(true);
				return;
			}
			setConnection(connection);

			if (
				connection.Role !== 'Source' ||
				(connection.Type !== 'App' && connection.Type !== 'Database' && connection.Type !== 'File')
			) {
				setHasImports(false);
				return;
			}

			// get the stats.
			let stats;
			[stats, err] = await call(`/api/connections/${connectionID}/stats`, 'GET');
			if (err) {
				onError(err);
				return;
			}
			let userStats = [];
			var ts = Math.round(new Date().getTime());
			for (let [i, userCount] of stats.UsersIn.entries()) {
				let relativeTs = ts + (i + 1) * 3600 * 1000;
				let d = new Date(relativeTs);
				let hour = d.getHours();
				userStats.push({ name: hour, users: userCount });
			}
			setUserStats(userStats);

			// get the imports.
			let imports;
			[imports, err] = await call('/admin/connections/imports', 'POST', connection.ID);
			if (err) {
				onError(err);
				return;
			}
			setImports(imports);
		};
		fetchConnection();
	}, []);

	const onImportConfirmation = async (e) => {
		let button = e.currentTarget;
		button.setAttribute('loading', '');
		let [, err] = await call('/admin/import-raw-user-data-from-connector', 'POST', {
			Connector: connectionID,
			ResetCursor: resetCursor,
		});
		button.removeAttribute('loading');
		if (err) {
			onError(err);
			setAskImportConfirmation(false);
			return;
		}
		setAskImportConfirmation(false);
		setStatus({ variant: 'primary', icon: 'cloud-download', text: 'Your import has been started' });
		toastRef.current.toast();
	};

	if (notFound) {
		return <NotFound />;
	}

	let cursorOptions = [
		{ Text: 'Pick up the import from where it left off', Icon: 'arrow-bar-right', Value: false },
		{ Text: 'Start importing all over again', Icon: 'arrow-clockwise', Value: true },
	];
	let c = connection;
	let tabs = [
		{ Name: 'Overview', Link: `/admin/connections/${c.ID}`, Selected: true },
		{ Name: 'Settings', Link: `/admin/connections/${c.ID}/settings`, Selected: false },
	];
	if (c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') {
		tabs.splice(1, 0, { Name: 'Properties', Link: `/admin/connections/${c.ID}/properties`, Selected: false });
	}
	if (c.Type === 'Database' && c.Role === 'Source') {
		tabs.splice(1, 0, { Name: 'SQL query', Link: `/admin/connections/${c.ID}/sql`, Selected: false });
	}
	return (
		<div className='ConnectionOverview'>
			<PrimaryBackground contentWidth={1400} height={300}>
				<Breadcrumbs
					onAccent={true}
					breadcrumbs={[{ Name: 'Connections', Link: '/admin/connections' }, { Name: `${c.Name}` }]}
				/>
				<ConnectionHeading connection={c} />
				<FlexContainer justifyContent='space-between'>
					<NavigationTabs tabs={tabs} onAccent={true} />
					{hasImports && (
						<SlButton
							className='importButton'
							variant='success'
							size='large'
							onClick={() => setAskImportConfirmation(true)}
						>
							<SlIcon slot='suffix' name='cloud-download' />
							Start a new import
						</SlButton>
					)}
				</FlexContainer>
			</PrimaryBackground>
			<div className='routeContent'>
				<Toast reactRef={toastRef} status={status} />
				{hasImports ? (
					<>
						<div className='chart'>
							<div className='title'>Users ingested by {c.Name} in the last 24 hours</div>
							<BarChart width={1400} height={350} data={userStats}>
								<CartesianGrid strokeDasharray='3 3' />
								<XAxis dataKey='name' />
								<YAxis />
								<Tooltip />
								<Bar dataKey='users' fill='var(--color-primary-600)' />
							</BarChart>
						</div>
						<div className='importsWrapper'>
							<div className='title'>Imports list</div>
							{imports.length > 0 ? (
								<div className='importsList'>
									<div className='headCell'>ID</div>
									<div className='headCell'>Start time</div>
									<div className='headCell'>End time</div>
									<div className='headCell'>Errors</div>
									{imports.map((i) => {
										return (
											<>
												<div class='cell'>{i.ID}</div>
												<div class='cell'>{i.StartTime}</div>
												<div class='cell'>{i.EndTime}</div>
												<div class='cell error'>{i.Error === '' ? '-' : i.Error}</div>
											</>
										);
									})}
								</div>
							) : (
								<div className='noImport'>
									No import has been yet performed from the {c.Name} connection
								</div>
							)}
						</div>
					</>
				) : (
					<div className='nothingToShow'>Currently there is nothing to show for connection {c.Name}</div>
				)}
			</div>
			<SlDialog
				open={askImportConfirmation}
				style={{ '--width': '600px' }}
				onSlAfterHide={() => setAskImportConfirmation(false)}
				className='askImportConfirmationDialog'
				label='Where do you want your import to start?'
			>
				<div className='resetCursorOptions'>
					{cursorOptions.map((opt) => {
						return (
							<div
								className={`resetCursorOption${opt.Value === resetCursor ? ' selected' : ''}`}
								onClick={() => setResetCursor(opt.Value)}
							>
								<SlIcon name={opt.Icon}></SlIcon>
								<div className='text'>{opt.Text}</div>
							</div>
						);
					})}
				</div>
				<SlButton variant='primary' size='large' onClick={onImportConfirmation}>
					<SlIcon slot='suffix' name='cloud-download' />
					Start import
				</SlButton>
			</SlDialog>
		</div>
	);
};

export default ConnectionOverview;
