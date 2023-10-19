import React, { useState, useRef, useContext, useEffect } from 'react';
import ConfirmationButton from '../../shared/ConfirmationButton/ConfirmationButton';
import Grid from '../../shared/Grid/Grid';
import Section from '../../shared/Section/Section';
import EditorWrapper from '../../shared/EditorWrapper/EditorWrapper';
import statuses from '../../../constants/statuses';
import * as variants from '../../../constants/variants';
import * as icons from '../../../constants/icons';
import { CONFIRM_ANIMATION_DURATION } from './Action.constants';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import ActionContext from '../../../context/ActionContext';
import { AppContext } from '../../../context/providers/AppProvider';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import { ExecQueryResponse } from '../../../types/external/api';

const queryMaxSize = 16777215;

const ActionQuery = () => {
	const [queryPreviewColumns, setQueryPreviewColumns] = useState<GridColumn[] | null>(null);
	const [queryPreviewRows, setQueryPreviewRows] = useState<GridRow[] | null>(null);
	const [showQueryPreviewContent, setShowQueryPreviewContent] = useState<boolean>(false);

	const { redirect, showError, showStatus, api } = useContext(AppContext);
	const { connection, action, setAction, actionType, setActionType, mappingSectionRef, setIsQueryChanged } =
		useContext(ActionContext);

	const queryConfirmButtonRef = useRef<any>();
	const lastQueryConfirmation = useRef('');

	useEffect(() => {
		lastQueryConfirmation.current = action.Query!;
	}, []);

	const onQueryPreview = async () => {
		const res = await query(20);
		if (res == null) {
			return;
		}
		const columns: GridColumn[] = [];
		for (const prop of res.Schema.properties!) {
			let name: string;
			if (prop.label != null && prop.label !== '') {
				name = prop.label;
			} else {
				name = prop.name;
			}
			columns.push({ name: name, type: prop.type.name });
		}
		const rows: GridRow[] = [];
		for (const row of res.Rows) {
			rows.push({ cells: Object.values(row) });
		}
		setQueryPreviewColumns(columns);
		setQueryPreviewRows(rows);
	};

	const onUpdateQuery = async (value: string | undefined) => {
		if (value == null) return;
		const a = { ...action };
		a.Query = value;
		setAction(a);
		if (value.trim() !== lastQueryConfirmation.current.trim() && lastQueryConfirmation.current.trim() !== '') {
			setIsQueryChanged(true);
		} else {
			setIsQueryChanged(false);
		}
	};

	const onConfirmQuery = async () => {
		queryConfirmButtonRef.current.load();
		const res = await query(0, true);
		if (res == null) {
			queryConfirmButtonRef.current.stop();
			return;
		}
		queryConfirmButtonRef.current.confirm();
		setTimeout(() => {
			const actionTyp = { ...actionType };
			actionTyp.InputSchema = res.Schema;
			setActionType(actionTyp);
			setTimeout(() => {
				const top = mappingSectionRef.current.getBoundingClientRect().top;
				mappingSectionRef.current.closest('.fullscreen').scrollBy({
					top: top - 130,
					left: 0,
					behavior: 'smooth',
				});
			});
		}, CONFIRM_ANIMATION_DURATION);
	};

	const query = async (limit: number, isConfirmation?: boolean) => {
		const a = { ...action };
		const trimmed = a.Query!.trim();
		if (trimmed.length > queryMaxSize) {
			showError('The query is too long');
			return;
		}
		if (!trimmed.includes('$limit')) {
			showError(`The query does not contain the $limit variable`);
			return;
		}
		let res: ExecQueryResponse;
		try {
			res = await api.workspaces.connections.query(connection.id, trimmed, limit);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'QueryExecutionFailed') {
					let statusMessage: string;
					if (err.cause && err.cause !== '') {
						statusMessage = err.cause;
					} else {
						statusMessage = err.message;
					}
					showStatus({ variant: variants.DANGER, icon: icons.CODE_ERROR, text: statusMessage });
				}
				return;
			}
			showError(err);
			return;
		}
		if (Object.keys(res.Schema.properties!).length === 0) {
			showError('The query execution did not yield any columns');
			return;
		}
		if (isConfirmation) {
			lastQueryConfirmation.current = trimmed;
		}
		return res;
	};

	return (
		<>
			<Section title='Query' description='The query used to import the data'>
				<EditorWrapper
					language='sql'
					height={400}
					name='actionQueryEditor'
					value={action.Query!}
					onChange={onUpdateQuery}
				></EditorWrapper>
				<div className='queryButtons'>
					<SlButton variant='neutral' size='small' onClick={onQueryPreview}>
						Preview
					</SlButton>
					<ConfirmationButton
						ref={queryConfirmButtonRef}
						variant='success'
						size='small'
						onClick={onConfirmQuery}
						animationDuration={CONFIRM_ANIMATION_DURATION}
					>
						Confirm
					</ConfirmationButton>
				</div>
			</Section>
			<SlDrawer
				className='previewDrawer'
				label='Query Preview'
				open={queryPreviewColumns != null && queryPreviewRows != null}
				onSlAfterShow={() => setShowQueryPreviewContent(true)}
				onSlAfterHide={() => {
					setQueryPreviewColumns(null);
					setQueryPreviewRows(null);
					setShowQueryPreviewContent(false);
				}}
				placement='bottom'
				style={{ '--size': '600px' } as React.CSSProperties}
			>
				{showQueryPreviewContent ? (
					<Grid
						columns={queryPreviewColumns!}
						rows={queryPreviewRows!}
						noRowsMessage={'Your query did not return data'}
					/>
				) : (
					<SlSpinner
						style={
							{
								fontSize: '3rem',
								'--track-width': '6px',
							} as React.CSSProperties
						}
					></SlSpinner>
				)}
			</SlDrawer>
		</>
	);
};

export default ActionQuery;
