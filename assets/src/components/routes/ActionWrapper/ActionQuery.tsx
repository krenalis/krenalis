import React, { useState, useRef, useContext, useEffect } from 'react';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import Grid from '../../base/Grid/Grid';
import Section from '../../base/Section/Section';
import EditorWrapper from '../../base/EditorWrapper/EditorWrapper';
import { CONFIRM_ANIMATION_DURATION, ERROR_ANIMATION_DURATION } from './Action.constants';
import { NotFoundError } from '../../../lib/api/errors';
import ActionContext from '../../../context/ActionContext';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { ExecQueryResponse } from '../../../lib/api/types/responses';
import { Popover } from '../../base/Popover/Popover';
import { ActionIssues } from './ActionIssues';

const queryMaxSize = 16777215;

const ActionQuery = () => {
	const [queryPreviewColumns, setQueryPreviewColumns] = useState<GridColumn[] | null>(null);
	const [queryPreviewRows, setQueryPreviewRows] = useState<GridRow[] | null>(null);
	const [showPreview, setShowPreview] = useState<boolean>(false);
	const [queryPreviewIssues, setQueryPreviewIssues] = useState<string[]>([]);

	const { redirect, handleError, api } = useContext(AppContext);
	const {
		connection,
		action,
		setAction,
		actionType,
		setActionType,
		transformationSectionRef,
		setIsQueryChanged,
		isTransformationDisabled,
		isEditing,
		setIssues,
		setShowIssues,
	} = useContext(ActionContext);

	const queryConfirmButtonRef = useRef<any>();
	const lastQueryConfirmation = useRef('');

	useEffect(() => {
		if (isEditing) {
			lastQueryConfirmation.current = action.query;
		}
	}, []);

	const onQueryPreview = async () => {
		setQueryPreviewIssues([]);
		const res = await query(20, false);
		if (res == null) {
			return;
		}
		setShowIssues(false);
		const columns: GridColumn[] = [];
		if (res.schema != null) {
			for (const prop of res.schema.properties!) {
				columns.push({ name: prop.name, type: prop.type.kind });
			}
		}
		const rows: GridRow[] = [];
		for (const row of res.rows) {
			const cells: any[] = [];
			for (const prop of res.schema.properties!) {
				const key = prop.name;
				cells.push(row[key]);
			}
			rows.push({ cells: cells });
		}
		setQueryPreviewColumns(columns);
		setQueryPreviewRows(rows);
		setQueryPreviewIssues(res.issues);
	};

	const onUpdateQuery = async (value: string | undefined) => {
		if (value == null) return;
		const a = { ...action };
		a.query = value;
		setAction(a);
		if (value.trim() !== lastQueryConfirmation.current.trim() && lastQueryConfirmation.current.trim() !== '') {
			setIsQueryChanged(true);
		} else {
			setIsQueryChanged(false);
		}
	};

	const onConfirmQuery = async () => {
		setIssues([]);
		queryConfirmButtonRef.current.load();
		const res = await query(0, true);
		if (res == null) {
			queryConfirmButtonRef.current.stop();
			return;
		}
		if (res.schema == null) {
			queryConfirmButtonRef.current.error("This query didn't return any compatible column");
			setTimeout(() => {
				setIssues(res.issues);
				const actionTyp = { ...actionType };
				actionTyp.inputSchema = null;
				setActionType(actionTyp);
			}, ERROR_ANIMATION_DURATION);
		} else {
			queryConfirmButtonRef.current.confirm();
			setTimeout(() => {
				setIssues(res.issues);
				const actionTyp = { ...actionType };
				actionTyp.inputSchema = res.schema;
				setActionType(actionTyp);
				setTimeout(() => {
					const top = transformationSectionRef.current.getBoundingClientRect().top;
					transformationSectionRef.current.closest('.fullscreen').scrollBy({
						top: top - 130,
						left: 0,
						behavior: 'smooth',
					});
				}, 100);
			}, CONFIRM_ANIMATION_DURATION);
		}
	};

	const query = async (limit: number, isConfirmation: boolean) => {
		const q = action.query!.trim();
		if (q.length > queryMaxSize) {
			handleError('The query is too long');
			return;
		}
		let res: ExecQueryResponse;
		try {
			res = await api.workspaces.connections.execQuery(connection.id, q, limit);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				handleError('The connection does not exist anymore');
				return;
			}
			handleError(err);
			return;
		}
		if (isConfirmation) {
			lastQueryConfirmation.current = q;
			setIsQueryChanged(false);
		}
		return res;
	};

	return (
		<>
			<Section
				title='Query'
				description='The query used to import the data.'
				className='action__query'
				padded={true}
				annotated={true}
			>
				<EditorWrapper
					language='sql'
					height={400}
					name='actionQueryEditor'
					value={action.query!}
					onChange={onUpdateQuery}
				></EditorWrapper>
				<div className='action__query-buttons'>
					<SlButton className='action__query-preview' variant='neutral' size='small' onClick={onQueryPreview}>
						Preview
					</SlButton>
					<FeedbackButton
						ref={queryConfirmButtonRef}
						className='action__query-confirm'
						variant='success'
						size='small'
						onClick={onConfirmQuery}
						animationDuration={CONFIRM_ANIMATION_DURATION}
					>
						Confirm
					</FeedbackButton>
					<Popover
						isOpen={isTransformationDisabled}
						content='Confirm when you have finished editing the query.'
					/>
				</div>
			</Section>
			<SlDrawer
				className='action__query-preview-drawer'
				open={queryPreviewColumns != null && queryPreviewRows != null}
				onSlAfterShow={() => setShowPreview(true)}
				onSlAfterHide={(e: any) => {
					if (e.target.classList.contains('action__issues')) {
						e.stopPropagation();
						return;
					}
					setQueryPreviewColumns(null);
					setQueryPreviewRows(null);
					setShowPreview(false);
					setShowIssues(true);
				}}
				placement='bottom'
				style={{ '--size': '600px' } as React.CSSProperties}
			>
				<div className='action__query-preview-drawer-label' slot='label'>
					<span>Query Preview</span>
					{showPreview && (
						<ActionIssues
							issues={queryPreviewIssues}
							type={connection.connector.type}
							role={connection.role}
						/>
					)}
				</div>
				{showPreview ? (
					<Grid
						columns={queryPreviewColumns!}
						rows={queryPreviewRows!}
						noRowsMessage={
							queryPreviewColumns.length === 0
								? "This query didn't return any compatible column"
								: 'This query did not return data'
						}
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
