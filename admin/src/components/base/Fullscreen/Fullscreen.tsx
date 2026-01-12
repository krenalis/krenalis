import React, { useState, ReactNode, useEffect } from 'react';
import './Fullscreen.css';
import { FullscreenContext } from '../../../context/FullscreenContext';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';

interface FullscreenProps {
	onClose: () => void;
	isLoading: boolean;
	className?: string;
	children: ReactNode;
}

const Fullscreen = ({ onClose, isLoading, className, children }: FullscreenProps) => {
	const [isOpen, setIsOpen] = useState(true);

	useEffect(() => {
		const html = document.querySelector('html');
		if (isOpen) {
			html.style.overflow = 'hidden';
		} else {
			html.style.overflow = 'auto';
		}
	}, [isOpen]);

	const onAnimationEnd = () => {
		if (!isOpen) {
			onClose();
		}
	};

	const closeFullscreen = (cb?: (...args: any) => void) => {
		setIsOpen(false);
		if (cb != null) {
			cb();
		}
	};

	return (
		<FullscreenContext.Provider value={{ closeFullscreen }}>
			<div
				className={`fullscreen${isOpen ? ' fullscreen--open' : ''}${isLoading ? ' fullscreen--loading' : ''}${className != null ? ' ' + className : ''}`}
				style={{ animation: `${isOpen ? 'fullscreenFadeIn' : 'fullscreenFadeOut'} .3s` }}
				onAnimationEnd={onAnimationEnd}
			>
				{isLoading ? (
					<SlSpinner
						style={
							{
								fontSize: '5rem',
								'--track-width': '6px',
							} as React.CSSProperties
						}
					/>
				) : (
					children
				)}
			</div>
		</FullscreenContext.Provider>
	);
};

export default Fullscreen;
