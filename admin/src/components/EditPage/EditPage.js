import { useContext } from 'react';
import './EditPage.css';
import { AppContext } from '../../context/AppContext';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const EditPage = ({ title, onCancel, children }) => {
	let { updateIsFullScreen } = useContext(AppContext);

	updateIsFullScreen(true);

	let onClose = () => {
		updateIsFullScreen(false);
		onCancel();
	};

	return (
		<div className='EditPage'>
			<div className='header'>
				<div className='title'>{title}</div>
				<SlButton variant='default' onClick={onClose}>
					<SlIcon name='x' slot='prefix'></SlIcon>
					Cancel
				</SlButton>
			</div>
			<div className='editPageContent'>{children}</div>
		</div>
	);
};

export default EditPage;
