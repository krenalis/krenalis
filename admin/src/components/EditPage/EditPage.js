import { useContext, useEffect } from 'react';
import './EditPage.css';
import { AppContext } from '../../context/AppContext';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const EditPage = ({ title, actions, onCancel, children }) => {
	let { updateIsFullScreen } = useContext(AppContext);

	useEffect(() => {
		const onPopState = () => window.location.reload();
		const onBeforeUnload = () => localStorage.removeItem('editPageHasBeenOpened');
		const cleanUp = () => {
			window.removeEventListener('popstate', onPopState);
			window.removeEventListener('beforeunload', onBeforeUnload);
		};

		window.addEventListener('popstate', onPopState);
		window.addEventListener('beforeunload', onBeforeUnload);

		let editPageHasAlreadyBeenOpened = localStorage.getItem('editPageHasBeenOpened');
		if (editPageHasAlreadyBeenOpened) {
			// avoid pushing the same history over and over if the user closes
			// and reopens the component.
			return cleanUp;
		}

		window.history.pushState(null, '', window.location);
		localStorage.setItem('editPageHasBeenOpened', true);

		return cleanUp;
	}, []);

	useEffect(() => {
		updateIsFullScreen(true);

		return () => {
			updateIsFullScreen(false);
		};
	}, []);

	return (
		<div className='EditPage'>
			<div className='header'>
				<div className='title'>{title}</div>
				<div className='actions'>
					<SlButton variant='default' onClick={onCancel}>
						<SlIcon name='x' slot='prefix'></SlIcon>
						Cancel
					</SlButton>
					{actions}
				</div>
			</div>
			<div className='editPageContent'>{children}</div>
		</div>
	);
};

export default EditPage;
