import React, { useState, useEffect, useContext } from 'react';
import './ConnectionOverview.css';
import Flex from '../../shared/Flex/Flex';
import Grid from '../../shared/Grid/Grid';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { NotFoundError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import { BarChart, Bar, XAxis, Tooltip, YAxis, CartesianGrid } from 'recharts';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import { ConnectionStats } from '../../../types/external/connection';
import { GridRow } from '../../../types/componentTypes/Grid.types';
import { Import } from '../../../types/external/api';

const IMPORTS_COLUMNS = [
	{ name: 'ID' },
	{ name: 'Start time', type: 'DateTime' },
	{ name: 'End time', type: 'DateTime' },
	{ name: 'Errors' },
];

const ConnectionOverview = () => {
	const [userStats, setUserStats] = useState<any[]>([]);
	const [imports, setImports] = useState<Import[]>([]);
	const [hasImports, setHasImports] = useState<boolean>(true);
	const [selectedError, setSelectedError] = useState<string>('');
	const [isLoading, setIsLoading] = useState<boolean>(true);

	const { api, showStatus, showError, redirect } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	useEffect(() => {
		const stopLoading = () => {
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		const fetchData = async () => {
			if (c.role !== 'Source' || (c.type !== 'App' && c.type !== 'Database' && c.type !== 'File')) {
				setHasImports(false);
				stopLoading();
				return;
			}
			// get the stats.
			let stats: ConnectionStats;
			try {
				stats = await api.workspace.connections.stats(c.id);
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
			const userStats: any[] = [];
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
			let imports: Import[];
			try {
				imports = await api.workspace.connections.imports(c.id);
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

	if (isLoading) {
		return (
			<div className='connectionOverview loading'>
				<SlSpinner
					style={
						{
							fontSize: '3rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				></SlSpinner>
			</div>
		);
	}

	const rows: GridRow[] = [];
	for (const i of imports) {
		const errorCell = (
			<div
				className={`cell error${i.Error !== '' ? ' hasError' : ''}`}
				onClick={() => {
					setSelectedError(i.Error);
				}}
			>
				{i.Error === '' ? '-' : i.Error}
			</div>
		);
		const row = { cells: [i.ID, i.StartTime, i.EndTime, errorCell], key: String(i.ID) };
		rows.push(row);
	}

	return (
		<div className='connectionOverview'>
			{hasImports ? (
				<>
					<div className='chart'>
						<Flex className='chartHead' justifyContent='space-between' alignItems='baseline'>
							<div className='title'>Users ingested by {c.name} in the last 24 hours</div>
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
						<Grid
							columns={IMPORTS_COLUMNS}
							rows={rows}
							noRowsMessage={`No import has been yet performed from the ${c.name} connection`}
						/>
					</div>
				</>
			) : (
				<div className='nothingToShow'>Currently there is nothing to show for connection {c.name}</div>
			)}
			<SlDialog
				open={selectedError !== ''}
				style={{ '--width': '600px' } as React.CSSProperties}
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
