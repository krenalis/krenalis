import React, { useContext, useState, useEffect, useLayoutEffect, ReactNode } from 'react';
import Grid from '../../base/Grid/Grid';
import { SCHEDULE_PERIODS } from '../../../lib/core/pipeline';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import SlRadio from '@shoelace-style/shoelace/dist/react/radio/index.js';
import SlRadioGroup from '@shoelace-style/shoelace/dist/react/radio-group/index.js';
import { Pipeline } from '../../../lib/api/types/pipeline';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import ConfirmByTyping from '../../base/ConfirmByTyping/ConfirmByTyping';
import { Variant } from '../App/App.types';
import { serializeFilter } from '../../../utils/filters';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';
import TransformedConnection from '../../../lib/core/connection';

const GRID_COLUMNS_WITH_FILTERS: GridColumn[] = [
	{ name: 'Pipeline' },
	{ name: 'Filters' },
	{ name: 'Enabled' },
	{ name: '' },
];
const GRID_COLUMNS_WITHOUT_FILTERS: GridColumn[] = [{ name: 'Pipeline' }, { name: 'Enabled' }, { name: '' }];

interface PipelinesGridProps {
	newPipelineID: React.MutableRefObject<number>;
	pipelines: Pipeline[];
	onSelectPipeline: (pipeline: Pipeline) => void;
}

const PipelinesGrid = ({ newPipelineID, pipelines, onSelectPipeline }: PipelinesGridProps) => {
	const [pipelineToDelete, setPipelineToDelete] = useState<Pipeline | null>();
	const [isAlertDialogOpen, setIsDialogAlertOpen] = useState<boolean>(false);
	const [deleteConfirmationInput, setDeleteConfirmationInput] = useState<string>('');
	const [windowWidth, setWindowWidth] = useState<number>(window.innerWidth);
	const [gridColumnsWidths, setGridColumnsWidths] = useState<string>();

	const {
		api,
		handleError,
		setIsLoadingConnections,
		connectors,
		runPipelineButtonRefs,
		runPipelineDropdownButtonRefs,
		runPipeline,
	} = useContext(AppContext);
	const { connection, setConnection } = useContext(ConnectionContext);

	useLayoutEffect(() => {
		const handleResize = () => {
			setWindowWidth(window.innerWidth);
		};
		handleResize();
		window.addEventListener('resize', handleResize);
		return () => window.removeEventListener('resize', handleResize);
	}, []);

	useLayoutEffect(() => {
		if (connection.isDatabase && connection.isSource) {
			if (windowWidth > 1700) {
				setGridColumnsWidths('280px 80px auto');
			} else if (windowWidth > 800) {
				setGridColumnsWidths('180px 80px auto');
			} else {
				setGridColumnsWidths('150px 80px auto');
			}
		} else {
			if (windowWidth > 1700) {
				setGridColumnsWidths('280px 320px 80px auto');
			} else if (windowWidth > 800) {
				setGridColumnsWidths('180px 220px 80px auto');
			} else {
				setGridColumnsWidths('150px 190px 80px auto');
			}
		}
	}, [windowWidth, connection.isDatabase, connection.isSource]);

	useEffect(() => {
		for (const a of pipelines) {
			if (runPipelineButtonRefs?.current[a.id]?.current == null) {
				runPipelineButtonRefs.current[a.id] = React.createRef();
			}
			if (runPipelineDropdownButtonRefs?.current[a.id]?.current == null) {
				runPipelineDropdownButtonRefs.current[a.id] = React.createRef();
			}
		}
	}, [pipelines]);

	const onPipelineStatusSwitch = async (pipelineID: number) => {
		const index = connection.pipelines!.findIndex((p) => p.id === pipelineID);
		const oldEnabled = connection.pipelines![index].enabled;
		const newEnabled = !oldEnabled;

		// Optimistically update the switch in the local state before awaiting
		// the API call. This ensures immediate UI feedback and prevents visible
		// flickering or delay caused by network latency. If the API request
		// fails, the previous value will be restored.
		const c = structuredClone(connection); // clone connection
		Object.setPrototypeOf(c, TransformedConnection.prototype); // restore class methods (structuredClone doesn't clone functions)
		c.pipelines![index].enabled = newEnabled;

		setConnection(c);

		try {
			await api.workspaces.connections.setPipelineStatus(pipelineID, newEnabled);
		} catch (err) {
			handleError(err);
			// Reset the switch
			c.pipelines![index].enabled = oldEnabled;
			setConnection(c);
			return;
		}
	};

	const onDeletePipeline = (pipelineID: number) => {
		setPipelineToDelete(pipelines.find((p) => p.id === pipelineID));
		setDeleteConfirmationInput('');
		setIsDialogAlertOpen(true);
	};

	const onConfirmDeletePipeline = async () => {
		newPipelineID.current = 0; // do not re-trigger the animation of the new pipeline's row during the repainting.
		try {
			await api.workspaces.connections.deletePipeline(pipelineToDelete.id);
		} catch (err) {
			handleError(err);
			closeAlertDialog();
			return;
		}
		closeAlertDialog();
		setTimeout(() => {
			setIsLoadingConnections(true);
		}, 300);
	};

	const closeAlertDialog = () => {
		setIsDialogAlertOpen(false);
		setDeleteConfirmationInput('');
		setTimeout(() => {
			// Reset the pipeline to delete after a delay to prevent flash of
			// content in the dialog, where the pipeline name is used.
			setPipelineToDelete(null);
		}, 300);
	};

	const onSchedulerPeriodChange = async (e: any, pipelineID: number) => {
		const period = e.currentTarget.value === 'Off' ? null : e.currentTarget.value;
		try {
			await api.workspaces.connections.setPipelineSchedulePeriod(pipelineID, period);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsLoadingConnections(true);
	};

	const onManageClick = (pipeline: Pipeline) => {
		for (const key in runPipelineButtonRefs.current) {
			const button = runPipelineButtonRefs.current[key].current;
			if (button != null) {
				button.hideTooltip();
			}
		}
		for (const key in runPipelineDropdownButtonRefs.current) {
			const button = runPipelineDropdownButtonRefs.current[key].current;
			if (button != null) {
				button.hideTooltip();
			}
		}
		onSelectPipeline(pipeline);
	};

	const onDropdownHide = (pipelineID: number) => {
		runPipelineButtonRefs.current[pipelineID].current.hideTooltip();
		runPipelineDropdownButtonRefs.current[pipelineID].current.hideTooltip();
	};

	const rows: GridRow[] = [];
	for (const pipeline of pipelines) {
		const pipelineType = connection.pipelineTypes.find(
			(t) => pipeline.target === t.target && (!('eventType' in pipeline) || pipeline.eventType === t.eventType),
		);
		if (pipelineType == null) {
			if (pipeline.target === 'Event' && connection.pipelineTypes.some((t) => t.target === 'Event')) {
				throw new Error(
					`Pipeline ${pipeline.id} has event type '${pipeline.eventType}', but the related connection does not have this event type`,
				);
			}
			throw new Error(
				`Pipeline ${pipeline.id} has target ${pipeline.target}, but the related connection ${connection.id} does not have this target`,
			);
		}

		const isPipelineRunSupported =
			(pipeline.target === 'User' || pipeline.target === 'Group') &&
			connection.connector.type !== 'SDK' &&
			connection.connector.type !== 'Webhook';

		let description = pipelineType.description;
		if (connection.isFileStorage) {
			if (connection.isSource) {
				description = `Import ${pipeline.target.toLowerCase()}s from "${pipeline.path}" into the data warehouse`;
			} else {
				description = `Export ${pipeline.target.toLowerCase()}s to "${pipeline.path}"`;
			}
		}

		let logo: ReactNode;
		if (connection.isFileStorage) {
			const formatConnector = connectors.find((c) => c.code === pipeline.format);
			logo = (
				<div className='connection-pipelines__pipeline-logo'>
					<span style={{ position: 'relative', top: '3px' }}>
						<LittleLogo code={formatConnector.code} path={CONNECTORS_ASSETS_PATH} />{' '}
					</span>
				</div>
			);
		}

		const nameCell = (
			<div className='connection-pipelines__pipeline-name'>
				{logo}
				<div className='connection-pipelines__pipeline-text'>
					<div className='connection-pipelines__pipeline-name-name'>{pipeline.name}</div>
					<div className='connection-pipelines__pipeline-name-description'>{description}</div>
				</div>
			</div>
		);

		let conditionsCell: ReactNode;
		if (pipeline.filter != null) {
			conditionsCell = (
				<div className='connection-pipelines__pipeline-filter'>{serializeFilter(pipeline.filter, true)}</div>
			);
		} else {
			conditionsCell = '-';
		}

		const enabledCell = (
			<SlSwitch onSlChange={() => onPipelineStatusSwitch(pipeline.id)} checked={pipeline.enabled}></SlSwitch>
		);

		let scheduleDotVariant: Variant = 'neutral';
		if (pipeline.enabled && pipeline.schedulePeriod != null) {
			scheduleDotVariant = 'success';
		}
		const pipelinesCell = (
			<div className='connection-pipelines__buttons'>
				{isPipelineRunSupported && (
					<>
						<FeedbackButton
							ref={runPipelineButtonRefs.current[pipeline.id]}
							className='connection-pipelines__run-button'
							size='small'
							onClick={() => runPipeline(connection, pipeline.id, pipeline.target)}
							disabled={!pipeline.enabled}
							hoist={true}
						>
							<SlIcon slot='prefix' name='play' />
							Run now
						</FeedbackButton>
						<SlDropdown hoist={true}>
							<SlButton
								slot='trigger'
								variant='default'
								size='small'
								className='connection-pipelines__scheduler-button'
							>
								<SlIcon slot='prefix' name='clock' />
								Schedule: {pipeline.schedulePeriod || 'Off'}
								<SlIcon
									slot='suffix'
									className={`connection-pipelines__scheduler-dot connection-pipelines__scheduler-dot--${scheduleDotVariant}`}
									name='circle-fill'
								/>
							</SlButton>
							<SlMenu className='connection-pipelines__scheduler-options'>
								<SlRadioGroup
									size='small'
									onSlChange={(e) => onSchedulerPeriodChange(e, pipeline.id)}
									value={pipeline.schedulePeriod || 'Off'}
								>
									{SCHEDULE_PERIODS.map((period) => (
										<SlRadio key={period} value={period}>
											{period}
										</SlRadio>
									))}
								</SlRadioGroup>
							</SlMenu>
						</SlDropdown>
					</>
				)}
				<SlButton
					className='connection-pipelines__manage-button'
					variant='default'
					size='small'
					onClick={() => onManageClick(pipeline)}
				>
					Manage...
				</SlButton>
				<SlDropdown
					className='connection-pipelines__buttons-dropdown'
					hoist={true}
					placement='bottom-end'
					onSlHide={() => onDropdownHide(pipeline.id)}
				>
					<SlButton
						slot='trigger'
						variant='default'
						size='small'
						className='connection-pipelines__menu-button'
					>
						<SlIcon slot='prefix' name='three-dots-vertical' />
					</SlButton>
					<SlMenu className='connection-pipelines__menu'>
						{isPipelineRunSupported && (
							<FeedbackButton
								ref={runPipelineDropdownButtonRefs.current[pipeline.id]}
								className='connection-pipelines__run-button'
								size='small'
								onClick={() => runPipeline(connection, pipeline.id, pipeline.target)}
								disabled={!pipeline.enabled}
								hoist={true}
								placement='left'
							>
								Run now
							</FeedbackButton>
						)}
						<SlMenuItem
							className='connection-pipelines__manage-button'
							onClick={() => onManageClick(pipeline)}
						>
							Manage...
						</SlMenuItem>
						<SlMenuItem
							className='connection-pipelines__delete'
							onClick={() => onDeletePipeline(pipeline.id)}
						>
							Delete
						</SlMenuItem>
					</SlMenu>
				</SlDropdown>
			</div>
		);

		const row: GridRow = {
			cells:
				connection.isDatabase && connection.isSource
					? [nameCell, enabledCell, pipelinesCell]
					: [nameCell, conditionsCell, enabledCell, pipelinesCell],
			key: String(pipeline.id),
		};
		if (pipeline.id === newPipelineID.current && connection.pipelines!.length > 1) {
			row.animation = 'fade-in';
		}

		rows.push(row);
	}

	return (
		<>
			<Grid
				className='connection-pipelines__grid'
				rows={rows}
				columns={
					connection.isDatabase && connection.isSource
						? GRID_COLUMNS_WITHOUT_FILTERS
						: GRID_COLUMNS_WITH_FILTERS
				}
				gridColumnsWidths={gridColumnsWidths}
				noRowsMessage='No pipelines to show'
			/>
			<AlertDialog
				variant='danger'
				isOpen={isAlertDialogOpen}
				onClose={closeAlertDialog}
				title={<span>Delete the pipeline?</span>}
				className='connection-pipelines__grid-alert'
				actions={
					<>
						<SlButton onClick={closeAlertDialog}>Cancel</SlButton>
						<SlButton
							variant='danger'
							onClick={onConfirmDeletePipeline}
							disabled={!pipelineToDelete || deleteConfirmationInput !== pipelineToDelete.name}
						>
							Delete pipeline
						</SlButton>
					</>
				}
			>
				{connection.isSource && pipelineToDelete?.target === 'User' ? (
					<p>
						{/* TODO Allineare a sx */}
						If you continue{' '}
						<span className='connection-pipelines__grid-alert-identities'>
							you will permanently lose the identities
						</span>{' '}
						imported by the pipeline. The profiles will be updated accordingly at the next identity
						resolution execution.
					</p>
				) : (
					<p>If you continue, you will permanently lose the pipeline</p>
				)}
				<ConfirmByTyping
					confirmText={pipelineToDelete?.name ?? ''}
					value={deleteConfirmationInput}
					onInput={setDeleteConfirmationInput}
				/>
			</AlertDialog>
		</>
	);
};

export default PipelinesGrid;
