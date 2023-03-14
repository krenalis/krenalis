import './EditPage.css';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const EditPage = ({ title, onCancel, children }) => {
	return (
		<div className='EditPage'>
			<div className='header'>
				<div className='title'>{title}</div>
				<SlButton variant='default' onClick={onCancel}>
					<SlIcon name='x' slot='prefix'></SlIcon>
					Cancel
				</SlButton>
			</div>
			<div className='editPageContent'>{children}</div>
		</div>
	);
};

export default EditPage;
