import React, { useContext, useState, useEffect, useLayoutEffect, ReactNode } from 'react';
import Grid from '../../base/Grid/Grid';
import { SCHEDULE_PERIODS } from '../../../lib/core/action';
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
import { Action } from '../../../lib/api/types/action';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import { Variant } from '../App/App.types';
import { serializeFilter } from '../../../utils/filters';
import LittleLogo from '../../base/LittleLogo/LittleLogo';

const GRID_COLUMNS: GridColumn[] = [{ name: 'Action' }, { name: 'Filter' }, { name: 'Enabled' }, { name: '' }];

interface ActionsGridProps {
	newActionID: React.MutableRefObject<number>;
	actions: Action[];
	onSelectAction: (action: Action) => void;
}

const ActionsGrid = ({ newActionID, actions, onSelectAction }: ActionsGridProps) => {
	const [actionToDelete, setActionToDelete] = useState<Action | null>();
	const [isAlertDialogOpen, setIsDialogAlertOpen] = useState<boolean>(false);
	const [windowWidth, setWindowWidth] = useState<number>(window.innerWidth);
	const [gridColumnsWidths, setGridColumnsWidths] = useState<string>();

	const {
		api,
		handleError,
		setIsLoadingConnections,
		connectors,
		executeActionButtonRefs,
		executeActionDropdownButtonRefs,
		executeAction,
	} = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	useLayoutEffect(() => {
		const handleResize = () => {
			setWindowWidth(window.innerWidth);
		};
		handleResize();
		window.addEventListener('resize', handleResize);
		return () => window.removeEventListener('resize', handleResize);
	}, []);

	useLayoutEffect(() => {
		if (windowWidth > 1700) {
			setGridColumnsWidths('280px 280px 80px auto');
		} else if (windowWidth > 800) {
			setGridColumnsWidths('180px 180px 80px auto');
		} else {
			setGridColumnsWidths('150px 150px 80px auto');
		}
	}, [windowWidth]);

	useEffect(() => {
		for (const a of actions) {
			if (executeActionButtonRefs?.current[a.id]?.current == null) {
				executeActionButtonRefs.current[a.id] = React.createRef();
			}
			if (executeActionDropdownButtonRefs?.current[a.id]?.current == null) {
				executeActionDropdownButtonRefs.current[a.id] = React.createRef();
			}
		}
	}, [actions]);

	const onActionStatusSwitch = async (actionID: number) => {
		const index = connection.actions!.findIndex((a) => a.id === actionID);
		const enabledValue = connection.actions![index].enabled;
		try {
			await api.workspaces.connections.setActionStatus(actionID, !enabledValue);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsLoadingConnections(true);
	};

	const onDeleteAction = (actionID: number) => {
		setActionToDelete(actions.find((a) => a.id === actionID));
		setIsDialogAlertOpen(true);
	};

	const onConfirmDeleteAction = async () => {
		newActionID.current = 0; // do not re-trigger the animation of the new action's row during the repainting.
		try {
			await api.workspaces.connections.deleteAction(actionToDelete.id);
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
		setTimeout(() => {
			// Reset the action to delete after a delay to prevent flash of
			// content in the dialog, where the action name is used.
			setActionToDelete(null);
		}, 300);
	};

	const onSchedulerPeriodChange = async (e: any, actionID: number) => {
		const period = e.currentTarget.value === 'Off' ? null : e.currentTarget.value;
		try {
			await api.workspaces.connections.setActionSchedulePeriod(actionID, period);
		} catch (err) {
			handleError(err);
			return;
		}
		setIsLoadingConnections(true);
	};

	const onManageClick = (action: Action) => {
		for (const key in executeActionButtonRefs.current) {
			const button = executeActionButtonRefs.current[key].current;
			if (button != null) {
				button.hideTooltip();
			}
		}
		for (const key in executeActionDropdownButtonRefs.current) {
			const button = executeActionDropdownButtonRefs.current[key].current;
			if (button != null) {
				button.hideTooltip();
			}
		}
		onSelectAction(action);
	};

	const onDropdownHide = (actionID: number) => {
		executeActionButtonRefs.current[actionID].current.hideTooltip();
		executeActionDropdownButtonRefs.current[actionID].current.hideTooltip();
	};

	const rows: GridRow[] = [];
	for (const action of actions) {
		const actionType = connection.actionTypes.find(
			(t) => action.target === t.target && (!('eventType' in action) || action.eventType === t.eventType),
		);
		if (actionType == null) {
			throw new Error(
				`Connection ${connection.id} no longer has target '${action.target}' and event type '${action.eventType}' for action ${action.id}`,
			);
		}

		const isActionExecutionSupported =
			(action.target === 'User' || action.target === 'Group') &&
			connection.connector.type !== 'SDK' &&
			connection.connector.type !== 'Webhook';

		let description = actionType.description;
		if (connection.isFileStorage) {
			if (connection.isSource) {
				description = `Import ${action.target.toLowerCase()}s from ${action.path} into the data warehouse`;
			} else {
				description = `Export ${action.target.toLowerCase()}s to ${action.path}`;
			}
		}

		let logo: ReactNode;
		if (connection.isFileStorage) {
			const formatConnector = connectors.find((c) => c.code === action.format);
			logo = (
				<div className='connection-actions__action-logo'>
					<span style={{ position: 'relative', top: '3px' }}>
						<LittleLogo code={formatConnector.code} />{' '}
					</span>
				</div>
			);
		}

		const nameCell = (
			<div className='connection-actions__action-name'>
				{logo}
				<div className='connection-actions__action-text'>
					<div className='connection-actions__action-name-name'>{action.name}</div>
					<div className='connection-actions__action-name-description'>{description}</div>
				</div>
			</div>
		);

		let conditionsCell: ReactNode;
		if (action.filter != null) {
			conditionsCell = (
				<div className='connection-actions__action-filter'>{serializeFilter(action.filter, true)}</div>
			);
		} else {
			conditionsCell = '-';
		}

		const enabledCell = (
			<SlSwitch onSlChange={() => onActionStatusSwitch(action.id)} checked={action.enabled}></SlSwitch>
		);

		let scheduleDotVariant: Variant = 'neutral';
		if (action.enabled && action.schedulePeriod != null) {
			scheduleDotVariant = 'success';
		}
		const actionsCell = (
			<div className='connection-actions__buttons'>
				{isActionExecutionSupported && (
					<>
						<FeedbackButton
							ref={executeActionButtonRefs.current[action.id]}
							className='connection-actions__run-button'
							size='small'
							onClick={() => executeAction(connection, action.id, action.target)}
							disabled={!action.enabled}
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
								className='connection-actions__scheduler-button'
							>
								<SlIcon slot='prefix' name='clock' />
								Schedule: {action.schedulePeriod || 'Off'}
								<SlIcon
									slot='suffix'
									className={`connection-actions__scheduler-dot connection-actions__scheduler-dot--${scheduleDotVariant}`}
									name='circle-fill'
								/>
							</SlButton>
							<SlMenu className='connection-actions__scheduler-options'>
								<SlRadioGroup
									size='small'
									onSlChange={(e) => onSchedulerPeriodChange(e, action.id)}
									value={action.schedulePeriod || 'Off'}
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
					className='connection-actions__manage-button'
					variant='default'
					size='small'
					onClick={() => onManageClick(action)}
				>
					Manage...
				</SlButton>
				<SlDropdown
					className='connection-actions__buttons-dropdown'
					hoist={true}
					placement='bottom-end'
					onSlHide={() => onDropdownHide(action.id)}
				>
					<SlButton slot='trigger' variant='default' size='small' className='connection-actions__menu-button'>
						<SlIcon slot='prefix' name='three-dots-vertical' />
					</SlButton>
					<SlMenu className='connection-actions__menu'>
						{isActionExecutionSupported && (
							<FeedbackButton
								ref={executeActionDropdownButtonRefs.current[action.id]}
								className='connection-actions__run-button'
								size='small'
								onClick={() => executeAction(connection, action.id, action.target)}
								disabled={!action.enabled}
								hoist={true}
								placement='left'
							>
								Run now
							</FeedbackButton>
						)}
						<SlMenuItem className='connection-actions__manage-button' onClick={() => onManageClick(action)}>
							Manage...
						</SlMenuItem>
						<SlMenuItem className='connection-actions__delete' onClick={() => onDeleteAction(action.id)}>
							Delete
						</SlMenuItem>
					</SlMenu>
				</SlDropdown>
			</div>
		);

		const row: GridRow = { cells: [nameCell, conditionsCell, enabledCell, actionsCell], key: String(action.id) };
		if (action.id === newActionID.current && connection.actions!.length > 1) {
			row.animation = 'fade-in';
		}

		rows.push(row);
	}

	return (
		<>
			<Grid
				className='connection-actions__grid'
				rows={rows}
				columns={GRID_COLUMNS}
				gridColumnsWidths={gridColumnsWidths}
				noRowsMessage='No actions to show'
			/>
			<AlertDialog
				variant='danger'
				isOpen={isAlertDialogOpen}
				onClose={closeAlertDialog}
				title={
					<span>
						Are you sure you want to delete the action{' '}
						<span className='connection-actions__grid-alert-action-name'>{actionToDelete?.name}</span> ?
					</span>
				}
				className='connection-actions__grid-alert'
				actions={
					<>
						<SlButton onClick={closeAlertDialog}>Cancel</SlButton>
						<SlButton variant='danger' onClick={onConfirmDeleteAction}>
							Delete
						</SlButton>
					</>
				}
			>
				<p>
					If you continue
					{connection.isSource && actionToDelete?.target.includes('User') && (
						<>
							{' '}
							<span className='connection-actions__grid-alert-identities'>
								you will permanently lose the identities
							</span>
						</>
					)}{' '}
					imported by the action. The user profiles will be updated accordingly at the next identity
					resolution execution.
				</p>
			</AlertDialog>
		</>
	);
};

export default ActionsGrid;
