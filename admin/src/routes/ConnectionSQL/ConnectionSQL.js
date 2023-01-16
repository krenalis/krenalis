import { useState } from 'react';
import './ConnectionSQL.css';
import Grid from '../../components/Grid/Grid';
import call from '../../utils/call';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';
import Editor from '@monaco-editor/react';

const queryMaxSize = 16777215;

const ConnectionSQL = ({ connection: c, onError, onStatusChange, isSelected }) => {
	let [query, setQuery] = useState(c.UsersQuery);
	let [limit, setLimit] = useState(20); // TODO(@Andrea): implement as a select
	let [table, setTable] = useState(null);

	const handlePreview = async () => {
		if (query.length > queryMaxSize) {
			onError('You query is too long');
			return;
		}
		if (!query.includes(':limit')) {
			onError(`Your query does not contain the ':limit' placeholder`);
			return;
		}
		let [table, err] = await call('/admin/connections/preview-query', 'POST', {
			Connection: c.ID,
			Query: query,
			Limit: limit,
		});
		if (err !== null) {
			onError(err);
			return;
		}
		if (table.Columns.length === 0) {
			onError('Your query did not return any columns');
			return;
		}
		if (table.Rows.length === 0) {
			onError('Your query did not return any rows');
			return;
		}
		setTable(table);
	};

	const saveQuery = async () => {
		if (query.length > queryMaxSize) {
			onError('You query is too long');
			return;
		}
		if (!query.includes(':limit')) {
			onError(`Your query does not contain the ':limit' placeholder`);
			return;
		}
		let [, err] = await call('/admin/connections/set-users-query', 'POST', {
			Connection: c.ID,
			Query: query,
		});
		if (err !== null) {
			onError(err);
			return;
		}
		onStatusChange({ variant: 'success', icon: 'check2-circle', text: 'Your query has been successfully saved' });
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
