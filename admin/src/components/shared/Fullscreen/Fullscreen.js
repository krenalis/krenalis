import { useState, useEffect } from 'react';
import './Fullscreen.css';

const Fullscreen = ({ isOpen, children }) => {
	const [render, setRender] = useState(false);

	useEffect(() => {
		const onPopState = () => window.location.reload();
		const onBeforeUnload = () => localStorage.removeItem('isFullscreen');
		const cleanUp = () => {
			window.removeEventListener('popstate', onPopState);
			window.removeEventListener('beforeunload', onBeforeUnload);
		};

		window.addEventListener('popstate', onPopState);
		window.addEventListener('beforeunload', onBeforeUnload);

		const isAlreadyFullscreen = localStorage.getItem('isFullscreen');
		if (isAlreadyFullscreen) {
			// avoid pushing the same history over and over if the user closes
			// and reopens the component.
			return cleanUp;
		}

		window.history.pushState(null, '', window.location);
		localStorage.setItem('editPageHasBeenOpened', true);

		return cleanUp;
	}, [isOpen]);

	useEffect(() => {
		if (isOpen) {
			setRender(true);
		}
	}, [isOpen]);

	const onAnimationEnd = () => {
		if (!isOpen) {
			setRender(false);
		}
	};

	return (
		render && (
			<div
				className={`fullscreen${isOpen ? ' isOpen' : ''}`}
				style={{ animation: `${isOpen ? 'fullscreenFadeIn' : 'fullscreenFadeOut'} .3s` }}
				onAnimationEnd={onAnimationEnd}
			>
				{children}
			</div>
		)
	);
};

export default Fullscreen;
