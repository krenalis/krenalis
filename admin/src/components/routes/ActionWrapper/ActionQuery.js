import { useState, useRef, useContext, useEffect } from 'react';
import ConfirmationButton from '../../shared/ConfirmationButton/ConfirmationButton';
import Grid from '../../shared/Grid/Grid';
import Section from '../../shared/Section/Section';
import EditorWrapper from '../../shared/EditorWrapper/EditorWrapper';
import * as statuses from '../../../constants/statuses';
import * as variants from '../../../constants/variants';
import * as icons from '../../../constants/icons';
import { CONFIRM_ANIMATION_DURATION } from './Action.constants';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { ActionContext } from '../../../context/ActionContext';
import { AppContext } from '../../../context/providers/AppProvider';
import { SlButton, SlSpinner, SlDrawer } from '@shoelace-style/shoelace/dist/react/index.js';

const queryMaxSize = 16777215;

const ActionQuery = () => {
	const [queryPreviewTable, setQueryPreviewTable] = useState(null);
	const [isQueryPreviewDrawerOpen, setIsQueryPreviewDrawerOpen] = useState(false);

	const { redirect, showError, showStatus, api } = useContext(AppContext);
	const { connection, action, setAction, actionType, setActionType, mappingSectionRef, setIsQueryChanged } =
		useContext(ActionContext);

	const queryConfirmButtonRef = useRef(null);
	const lastQueryConfirmation = useRef('');

	useEffect(() => {
		lastQueryConfirmation.current = action.Query;
	}, []);

	const onQueryPreview = async () => {
		const res = await query(20);
		if (res == null) {
			return;
		}
		const columns = [];
		for (const prop of res.Schema.properties) {
			let name;
			if (prop.label != null && prop.label !== '') {
				name = prop.label;
			} else {
				name = prop.name;
			}
			columns.push({ name: name, type: prop.type.name });
		}
		const rows = [];
		for (const row of res.Rows) {
			rows.push({ cells: row });
		}
		const table = { columns, rows };
		setQueryPreviewTable(table);
	};

	const onUpdateQuery = async (value) => {
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

	const query = async (limit, isConfirmation) => {
		const a = { ...action };
		const trimmed = a.Query.trim();
		if (trimmed.length > queryMaxSize) {
			showError('You query is too long');
			return;
		}
		if (!trimmed.includes('$limit')) {
			showError(`Your query does not contain the $limit variable`);
			return;
		}
		const [res, err] = await api.connections.query(connection.id, trimmed, limit);
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'QueryExecutionFailed') {
					let statusMessage;
					if (err.cause && err.cause !== '') {
						statusMessage = err.cause;
					} else {
						statusMessage = err.message;
					}
					showStatus([variants.DANGER, icons.CODE_ERROR, statusMessage]);
				}
				return;
			}
			showError(err);
			return;
		}
		if (Object.keys(res.Schema.properties).length === 0) {
			showError('Your query did not return any columns');
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
					defaultLanguage='sql'
					height={400}
					value={action.Query}
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
				open={queryPreviewTable != null}
				onSlAfterShow={() => setIsQueryPreviewDrawerOpen(true)}
				onSlAfterHide={() => {
					setQueryPreviewTable(null);
					setIsQueryPreviewDrawerOpen(false);
				}}
				placement='bottom'
				style={{ '--size': '600px' }}
			>
				{isQueryPreviewDrawerOpen ? (
					<Grid
						columns={queryPreviewTable.columns}
						rows={queryPreviewTable.rows}
						noRowsMessage={'Your query did not return data'}
					/>
				) : (
					<SlSpinner
						style={{
							fontSize: '3rem',
							'--track-width': '6px',
						}}
					></SlSpinner>
				)}
			</SlDrawer>
		</>
	);
};

export default ActionQuery;
