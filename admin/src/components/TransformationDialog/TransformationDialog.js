import './TransformationDialog.css';
import { getTransformationType } from '../../utils/getTransformationType';
import { SlDialog, SlButton } from '@shoelace-style/shoelace/dist/react/index.js';
import Editor from '@monaco-editor/react';

const TransformationDialog = ({ transformation: t, onClose, onEditorChange, onRemove }) => {
	let transformationType = getTransformationType(t);
	let dialog;
	if (transformationType === 'one-to-one') {
		return null;
	} else if (transformationType === 'predefined') {
		dialog = null;
	} else if (transformationType === 'custom') {
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
