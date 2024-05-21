import React, { useState, useRef, useContext, useEffect } from 'react';
import FeedbackButton from '../../shared/FeedbackButton/FeedbackButton';
import Grid from '../../shared/Grid/Grid';
import Section from '../../shared/Section/Section';
import EditorWrapper from '../../shared/EditorWrapper/EditorWrapper';
import { CONFIRM_ANIMATION_DURATION } from './Action.constants';
import { NotFoundError } from '../../../lib/api/errors';
import ActionContext from '../../../context/ActionContext';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';
import { GridColumn, GridRow } from '../../shared/Grid/Grid.types';
import { ExecQueryResponse } from '../../../types/external/api';
import { Popover } from '../../shared/Popover/Popover';

const queryMaxSize = 16777215;

const ActionQuery = () => {
	const [queryPreviewColumns, setQueryPreviewColumns] = useState<GridColumn[] | null>(null);
	const [queryPreviewRows, setQueryPreviewRows] = useState<GridRow[] | null>(null);
	const [showPreview, setShowPreview] = useState<boolean>(false);

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
	} = useContext(ActionContext);

	const queryConfirmButtonRef = useRef<any>();
	const lastQueryConfirmation = useRef('');

	useEffect(() => {
		if (isEditing) {
			lastQueryConfirmation.current = action.Query;
		}
	}, []);

	const onQueryPreview = async () => {
		const res = await query(20, false);
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
			const cells: any[] = [];
			for (const prop of res.Schema.properties!) {
				const key = prop.name;
				cells.push(row[key]);
			}
			rows.push({ cells: cells });
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
				const top = transformationSectionRef.current.getBoundingClientRect().top;
				transformationSectionRef.current.closest('.fullscreen').scrollBy({
					top: top - 130,
					left: 0,
					behavior: 'smooth',
				});
			}, 100);
		}, CONFIRM_ANIMATION_DURATION);
	};

	const query = async (limit: number, isConfirmation: boolean) => {
		const q = action.Query!.trim();
		if (q.length > queryMaxSize) {
			handleError('The query is too long');
			return;
		}
		if (!/\${\s*limit\s*}/i.test(q)) {
			handleError('The query does not contain the ${limit} placeholder');
			return;
		}
		let res: ExecQueryResponse;
		try {
			res = await api.workspaces.connections.query(connection.id, q, limit);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				handleError('The connection does not exist anymore');
				return;
			}
			handleError(err);
			return;
		}
		if (res.Schema.properties!.length === 0) {
			handleError('The query execution did not yield any columns');
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
				description='The query used to import the data. It must contain the placeholder ${limit}.'
				className='action__query'
			>
				<EditorWrapper
					language='sql'
					height={400}
					name='actionQueryEditor'
					value={action.Query!}
					onChange={onUpdateQuery}
				></EditorWrapper>
				<div className='action__query-buttons'>
					<SlButton variant='neutral' size='small' onClick={onQueryPreview}>
						Preview
					</SlButton>
					<FeedbackButton
						ref={queryConfirmButtonRef}
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
				label='Query Preview'
				open={queryPreviewColumns != null && queryPreviewRows != null}
				onSlAfterShow={() => setShowPreview(true)}
				onSlAfterHide={() => {
					setQueryPreviewColumns(null);
					setQueryPreviewRows(null);
					setShowPreview(false);
				}}
				placement='bottom'
				style={{ '--size': '600px' } as React.CSSProperties}
			>
				{showPreview ? (
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
