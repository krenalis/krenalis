import './TransformationDialog.css';
import { SlDialog, SlButton } from '@shoelace-style/shoelace/dist/react/index.js';
import Editor from '@monaco-editor/react';

const TransformationDialog = ({ transformation: t, onClose, onEditorChange, onRemove }) => {
	let dialog;
	if (t.Type === 'one-to-one') {
		return null;
	} else if (t.Type === 'predefined') {
		dialog = null;
	} else if (t.Type === 'custom') {
		dialog = (
			<SlDialog
				label='Modify the transformation'
				open={true}
				onSlAfterHide={onClose}
				style={{ '--width': '700px' }}
			>
				<div className='editorWrapper'>
					<Editor
						onChange={onEditorChange}
						defaultLanguage='python'
						value={t.CustomFunc.Source}
						theme='vs-light'
					/>
				</div>
				<SlButton className='removeTransformation' slot='footer' variant='danger' onClick={onRemove}>
					Remove
				</SlButton>
			</SlDialog>
		);
	}
	return dialog;
};

export default TransformationDialog;
