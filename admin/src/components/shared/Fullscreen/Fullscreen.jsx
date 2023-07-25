import { useState } from 'react';
import './Fullscreen.css';
import { FullscreenContext } from '../../../context/FullscreenContext';
import { SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

const Fullscreen = ({ onClose, isLoading, children }) => {
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
				className={`fullscreen${isOpen ? ' isOpen' : ''}${isLoading ? ' isLoading' : ''}`}
				style={{ animation: `${isOpen ? 'fullscreenFadeIn' : 'fullscreenFadeOut'} .3s` }}
				onAnimationEnd={onAnimationEnd}
			>
				{isLoading ? (
					<SlSpinner
						style={{
							fontSize: '5rem',
							'--track-width': '6px',
						}}
					/>
				) : (
					children
				)}
			</div>
		</FullscreenContext.Provider>
	);
};

export default Fullscreen;
