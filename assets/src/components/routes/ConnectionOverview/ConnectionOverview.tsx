import React, { useState, useEffect, useContext } from 'react';
import './ConnectionOverview.css';
import Flex from '../../shared/Flex/Flex';
import Grid from '../../shared/Grid/Grid';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import { NotFoundError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import { BarChart, Bar, XAxis, Tooltip, YAxis, CartesianGrid } from 'recharts';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import { ConnectionStats } from '../../../types/external/connection';
import { GridRow } from '../../../types/componentTypes/Grid.types';
import { Execution } from '../../../types/external/api';

const EXECUTIONS_COLUMNS = [
	{ name: 'ID' },
	{ name: 'Start time', type: 'DateTime' },
	{ name: 'End time', type: 'DateTime' },
	{ name: 'Errors' },
];

const ConnectionOverview = () => {
	const [userStats, setUserStats] = useState<any[]>([]);
	const [executions, setExecutions] = useState<Execution[]>([]);
	const [hasExecutions, setHasExecutions] = useState<boolean>(true);
	const [selectedExecution, setSelectedExecution] = useState<Execution>(null);
	const [isLoading, setIsLoading] = useState<boolean>(true);

	const { api, showStatus, handleError, redirect } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	useEffect(() => {
		const stopLoading = () => {
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		const fetchData = async () => {
			if (c.type !== 'App' && c.type !== 'Database' && c.type !== 'FileStorage') {
				setHasExecutions(false);
				stopLoading();
				return;
			}
			// get the stats.
			let stats: ConnectionStats;
			try {
				stats = await api.workspaces.connections.stats(c.id);
			} catch (err) {
				if (err instanceof NotFoundError) {
					redirect('connections');
					showStatus(statuses.connectionDoesNotExistAnymore);
					stopLoading();
					return;
				}
				handleError(err);
				stopLoading();
				return;
			}
			const userStats: any[] = [];
			// compute the last 24 hours.
			var ts = Math.round(new Date().getTime());
			for (const [i, userCount] of stats.UserIdentities.entries()) {
				const relativeTs = ts + (i + 1) * 3600 * 1000;
				const d = new Date(relativeTs);
				const hour = d.getHours();
				userStats.push({ name: hour, users: userCount });
			}
			setUserStats(userStats);

			// get the executions.
			let executions: Execution[];
			try {
				executions = await api.workspaces.connections.executions(c.id);
			} catch (err) {
				handleError(err);
				stopLoading();
				return;
			}
			setExecutions(executions);
			stopLoading();
			const params = new URLSearchParams(window.location.search);
			const hasFailedExecution = params.has('failed-execution-action');
			if (hasFailedExecution) {
				setTimeout(() => {
					const executionsListElement = document.querySelector('.connection-overview__executions');
					executionsListElement.scrollIntoView({ behavior: 'smooth' });
				}, 500);
			}
		};
		fetchData();
	}, []);

	if (isLoading) {
		return (
			<div className='connection-overview--loading'>
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
	for (const exec of executions) {
		const errorCell = (
			<div
				className={`connection-overview__cell-error${exec.Error !== '' ? ' connection-overview__cell-error--has-error' : ''}`}
				onClick={() => {
					setSelectedExecution(exec);
				}}
			>
				{exec.Error === '' ? '-' : exec.Error}
			</div>
		);
		const row = { cells: [exec.ID, exec.StartTime, exec.EndTime, errorCell], key: String(exec.ID) };
		rows.push(row);
	}

	return (
		<div className='connection-overview'>
			{hasExecutions ? (
				<>
					{c.role === 'Source' && (
						<div className='connection-overview__chart'>
							<Flex
								className='connection-overview__cart-head'
								justifyContent='space-between'
								alignItems='baseline'
							>
								<div className='connection-overview__cart-title'>
									User identities ingested by {c.name} in the last 24 hours
								</div>
							</Flex>
							<BarChart width={1400} height={350} data={userStats}>
								<CartesianGrid strokeDasharray='3 3' />
								<XAxis dataKey='name' />
								<YAxis />
								<Tooltip />
								<Bar dataKey='users' fill='var(--color-primary-600)' />
							</BarChart>
						</div>
					)}
					<div className='connection-overview__executions'>
						<div className='connection-overview__executions-title'>
							{c.role === 'Source' ? 'Imports' : 'Exports'}
						</div>
						<Grid
							columns={EXECUTIONS_COLUMNS}
							rows={rows}
							noRowsMessage={`No execution has been yet performed from the ${c.name} connection`}
						/>
					</div>
				</>
			) : (
				<div className='connection-overview__nothing-to-show'>
					Currently there is nothing to show for connection {c.name}
				</div>
			)}
			<SlDialog
				open={selectedExecution !== null}
				style={{ '--width': '600px' } as React.CSSProperties}
				onSlAfterHide={() => setSelectedExecution(null)}
				className='connection-overview__selected-error-dialog'
				label={selectedExecution ? `${c.role === 'Source' ? 'Import' : 'Export'} ${selectedExecution.ID}` : ''}
			>
				{selectedExecution ? selectedExecution.Error : ''}
			</SlDialog>
		</div>
	);
};

export default ConnectionOverview;
