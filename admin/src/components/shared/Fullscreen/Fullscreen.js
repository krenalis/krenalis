import { useState } from 'react';
import { FullscreenContext } from '../../../context/FullscreenContext';
import './Fullscreen.css';

const Fullscreen = ({ onClose, children }) => {
	const [isOpen, setIsOpen] = useState(true);

	const onAnimationEnd = () => {
		if (!isOpen) {
			onClose();
		}
	};

	const closeFullscreen = () => {
		setIsOpen(false);
	};

	return (
		<FullscreenContext.Provider value={{ closeFullscreen }}>
			<div
				className={`fullscreen${isOpen ? ' isOpen' : ''}`}
				style={{ animation: `${isOpen ? 'fullscreenFadeIn' : 'fullscreenFadeOut'} .3s` }}
				onAnimationEnd={onAnimationEnd}
			>
				{children}
			</div>
		</FullscreenContext.Provider>
	);
};

export default Fullscreen;
