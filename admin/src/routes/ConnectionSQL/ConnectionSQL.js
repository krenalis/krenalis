import { useState, useContext } from 'react';
import './ConnectionSQL.css';
import Grid from '../../components/Grid/Grid';
import { NotFoundError, UnprocessableError } from '../../api/errors';
import { AppContext } from '../../context/AppContext';
import statuses from '../../constants/statuses';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';
import Editor from '@monaco-editor/react';

const queryMaxSize = 16777215;

const ConnectionSQL = ({ connection: c, isSelected }) => {
	let [query, setQuery] = useState(c.UsersQuery);
	let [limit, setLimit] = useState(20); // TODO(@Andrea): implement as a select
	let [table, setTable] = useState(null);

	const { API, showError, showStatus, redirect } = useContext(AppContext);

	const handlePreview = async () => {
		if (query.length > queryMaxSize) {
			showError('You query is too long');
			return;
		}
		if (!query.includes(':limit')) {
			showError(`Your query does not contain the ':limit' placeholder`);
			return;
		}
		let [table, err] = await API.connections.query(c.ID, query, limit);
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'QueryExecutionFailed') {
					showStatus(statuses.queryExecutionFailed);
				}
				return;
			}
			showError(err);
			return;
		}
		if (table.Columns.length === 0) {
			showError('Your query did not return any columns');
			return;
		}
		if (table.Rows.length === 0) {
			showError('Your query did not return any rows');
			return;
		}
		setTable(table);
	};

	const saveQuery = async () => {
		if (query.length > queryMaxSize) {
			showError('You query is too long');
			return;
		}
		if (!query.includes(':limit')) {
			showError(`Your query does not contain the ':limit' placeholder`);
			return;
		}
		let [, err] = await API.connections.setUsersQuery(c.ID, query);
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			showError(err);
			return;
		}
		showStatus(statuses.querySet);
	};

	return (
		<>
			<div className='ConnectionSQL'>
				<div className='editorWrapper'>
					<Editor
						onChange={(value) => setQuery(value)}
						defaultLanguage='sql'
						value={query}
						theme='vs-primary'
					/>
				</div>
				<div className='buttons'>
					<SlButton className='previewButton' variant='neutral' size='large' onClick={handlePreview}>
						<SlIcon slot='prefix' name='eye' />
						Preview
					</SlButton>
					<SlButton className='saveButton' variant='primary' size='large' onClick={saveQuery}>
						<SlIcon slot='prefix' name='save' />
						Save
					</SlButton>
				</div>
			</div>
			{table && (
				<SlDialog
					label='Users preview'
					open={true}
					style={{ '--width': '1200px' }}
					onSlAfterHide={() => setTable(null)}
				>
					<Grid table={table} />
				</SlDialog>
			)}
		</>
	);
};

export default ConnectionSQL;
