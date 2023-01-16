import { useState, useEffect } from 'react';
import './ConnectionOverview.css';
import call from '../../utils/call';
import FlexContainer from '../../components/FlexContainer/FlexContainer';
import { BarChart, Bar, XAxis, Tooltip, YAxis, CartesianGrid } from 'recharts';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionOverview = ({ connection: c, onError, onStatusChange, isSelected }) => {
	let [userStats, setUserStats] = useState([]);
	let [imports, setImports] = useState([]);
	let [hasImports, setHasImports] = useState(true);
	let [askImportConfirmation, setAskImportConfirmation] = useState(false);
	let [selectedError, setSelectedError] = useState('');
	let [resetCursor, setResetCursor] = useState(false);

	useEffect(() => {
		const fetchData = async () => {
			if (c.Role !== 'Source' || (c.Type !== 'App' && c.Type !== 'Database' && c.Type !== 'File')) {
				setHasImports(false);
				return;
			}
			let err;
			// get the stats.
			let stats;
			[stats, err] = await call(`/api/connections/${c.ID}/stats`, 'GET');
			if (err) {
				onError(err);
				return;
			}
			let userStats = [];
			// compute the last 24 hours.
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
			[imports, err] = await call('/admin/connections/imports', 'POST', c.ID);
			if (err) {
				onError(err);
				return;
			}
			setImports(imports);
		};
		fetchData();
	}, []);

	const onImportConfirmation = async (e) => {
		let button = e.currentTarget;
		button.setAttribute('loading', '');
		let [, err] = await call('/admin/import-raw-user-data-from-connector', 'POST', {
			Connector: c.ID,
			ResetCursor: resetCursor,
		});
		button.removeAttribute('loading');
		if (err) {
			onError(err);
			setAskImportConfirmation(false);
			return;
		}
		setAskImportConfirmation(false);
		onStatusChange({ variant: 'primary', icon: 'cloud-download', text: 'Your import has been started' });
	};

	let cursorOptions = [
		{ Text: 'Pick up the import from where it left off', Icon: 'arrow-bar-right', Value: false },
		{ Text: 'Start importing all over again', Icon: 'arrow-clockwise', Value: true },
	];
	return (
		<div className='ConnectionOverview'>
			{hasImports ? (
				<>
					<div className='chart'>
						<FlexContainer className='chartHead' justifyContent='space-between' alignItems='baseline'>
							<div className='title'>Users ingested by {c.Name} in the last 24 hours</div>
							<SlButton
								className='importButton'
								variant='primary'
								size='large'
								onClick={() => setAskImportConfirmation(true)}
							>
								<SlIcon slot='suffix' name='cloud-download' />
								Start a new import
							</SlButton>
						</FlexContainer>
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
											<div
												class={`cell error${i.Error !== '' ? ' hasError' : ''}`}
												onClick={() => {
													setSelectedError(i.Error);
												}}
											>
												{i.Error === '' ? '-' : i.Error}
											</div>
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
			<SlDialog
				open={selectedError !== ''}
				style={{ '--width': '600px' }}
				onSlAfterHide={() => setSelectedError('')}
				className='selectedErrorDialog'
				label='Full length error'
			>
				{selectedError}
			</SlDialog>
		</div>
	);
};

export default ConnectionOverview;
