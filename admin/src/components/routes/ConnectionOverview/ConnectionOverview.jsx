import { useState, useEffect, useContext } from 'react';
import './ConnectionOverview.css';
import Flex from '../../shared/Flex/Flex';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import { BarChart, Bar, XAxis, Tooltip, YAxis, CartesianGrid } from 'recharts';
import { SlButton, SlIcon, SlDialog, SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionOverview = () => {
	const [userStats, setUserStats] = useState([]);
	const [imports, setImports] = useState([]);
	const [hasImports, setHasImports] = useState(true);
	const [askImportConfirmation, setAskImportConfirmation] = useState(false);
	const [selectedError, setSelectedError] = useState('');
	const [resetCursor, setResetCursor] = useState(false);
	const [isLoading, setIsLoading] = useState(true);

	const { api, showStatus, showError, redirect } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	useEffect(() => {
		const stopLoading = () => {
			setTimeout(() => {
				setIsLoading(false);
			}, 500);
		};
		const fetchData = async () => {
			if (c.role !== 'Source' || (c.type !== 'App' && c.type !== 'Database' && c.type !== 'File')) {
				setHasImports(false);
				stopLoading();
				return;
			}
			let err;
			// get the stats.
			let stats;
			try {
				stats = await api.connections.stats(c.id);
			} catch (err) {
				if (err instanceof NotFoundError) {
					redirect('connections');
					showStatus(statuses.connectionDoesNotExistAnymore);
					stopLoading();
					return;
				}
				showError(err);
				stopLoading();
				return;
			}
			const userStats = [];
			// compute the last 24 hours.
			var ts = Math.round(new Date().getTime());
			for (const [i, userCount] of stats.Users.entries()) {
				const relativeTs = ts + (i + 1) * 3600 * 1000;
				const d = new Date(relativeTs);
				const hour = d.getHours();
				userStats.push({ name: hour, users: userCount });
			}
			setUserStats(userStats);

			// get the imports.
			let imports;
			try {
				imports = await api.connections.imports(c.id);
			} catch (err) {
				showError(err);
				stopLoading();
				return;
			}
			setImports(imports);
			stopLoading();
		};
		fetchData();
	}, []);

	// TODO: 'api.connections.import' doesn't exist anymore. The page isn't
	// working currently and should be redesigned from scratch.
	const onImportConfirmation = async (e) => {
		// const button = e.currentTarget;
		// button.setAttribute('loading', '');
		// const [, err] = await api.connections.import(c.id, resetCursor);
		// button.removeAttribute('loading');
		// if (err) {
		// 	if (err instanceof NotFoundError) {
		// 		redirect('connections');
		// 		showStatus(statuses.connectionDoesNotExistAnymore);
		// 		return;
		// 	}
		// 	if (err instanceof UnprocessableError) {
		// 		switch (err.code) {
		// 			case 'AlreadyInProgress':
		// 				showStatus(statuses.alreadyInProgress);
		// 				break;
		// 			case 'NoStorage':
		// 				showStatus(statuses.noStorage);
		// 				break;
		// 			case 'NoTransformationNorMappings':
		// 				showStatus(statuses.noTransformationNorMappings);
		// 				break;
		// 			case 'NoWarehouse':
		// 				showStatus(statuses.noWarehouse);
		// 				break;
		// 			case 'NotEnabled':
		// 				showStatus(statuses.notEnabled);
		// 				break;
		// 			case 'StorageNotEnabled':
		// 				showStatus(statuses.storageNotEnabled);
		// 				break;
		// 			default:
		// 				break;
		// 		}
		// 		return;
		// 	}
		// 	showError(err);
		// 	setAskImportConfirmation(false);
		// 	return;
		// }
		// setAskImportConfirmation(false);
		// showStatus(statuses.importStarted);
	};

	if (isLoading) {
		return (
			<div className='ConnectionOverview loading'>
				<SlSpinner
					style={{
						fontSize: '3rem',
						'--track-width': '6px',
					}}
				></SlSpinner>
			</div>
		);
	}

	const cursorOptions = [
		{ Text: 'Pick up the import from where it left off', Icon: 'arrow-bar-right', Value: false },
		{ Text: 'Start importing all over again', Icon: 'arrow-clockwise', Value: true },
	];
	return (
		<div className='connectionOverview'>
			{hasImports ? (
				<>
					<div className='chart'>
						<Flex className='chartHead' justifyContent='space-between' alignItems='baseline'>
							<div className='title'>Users ingested by {c.name} in the last 24 hours</div>
							<SlButton
								className='importButton'
								variant='text'
								onClick={() => setAskImportConfirmation(true)}
							>
								Start a new import...
							</SlButton>
						</Flex>
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
								No import has been yet performed from the {c.name} connection
							</div>
						)}
					</div>
				</>
			) : (
				<div className='nothingToShow'>Currently there is nothing to show for connection {c.name}</div>
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
