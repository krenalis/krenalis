import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import './ConfirmationButton.css';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

const ConfirmationButton = forwardRef(({ children, className, animationDuration, ...delegated }, ref) => {
	let [isLoading, setIsLoading] = useState(false);

	let buttonRef = useRef(null);
	let loadingDurationRef = useRef(0);

	const duration = animationDuration || 1200;
	const halfAnimation = duration / 2;

	let isAnimating = false;
	const confirmAnimation = () => {
		if (isAnimating) {
			return;
		}
		isAnimating = true;
		let button = buttonRef.current;
		let initialText = button.textContent;
		let initialWidth = button.getBoundingClientRect().width;

		button.classList.add('confirm');
		button.style.width = `${initialWidth}px`;

		let icon = `<sl-icon name='check'></sl-icon>`;
		button.innerHTML = icon;

		setTimeout(() => {
			button.classList.remove('confirm');
			setTimeout(() => {
				button.innerHTML = initialText;
				button.style.width = `unset`;
				isAnimating = false;
			}, halfAnimation);
		}, halfAnimation);
	};

	const startLoading = () => {
		loadingDurationRef.current = Date.now();
		setIsLoading(true);
	};

	const stopLoading = (cb) => {
		let now = Date.now();
		let delta = now - loadingDurationRef.current;
		if (delta < 500) {
			// setta il timeout prima di stoppare lo spinner
			setTimeout(() => {
				setIsLoading(false);
				if (cb) cb();
			}, 500);
		} else {
			setIsLoading(false);
			if (cb) cb();
		}
	};

	useImperativeHandle(
		ref,
		() => ({
			confirm() {
				stopLoading(confirmAnimation);
			},
			load() {
				startLoading();
			},
			stop() {
				stopLoading();
			},
		}),
		[]
	);

	return (
		<SlButton
			className={`confirmationButton${className != null ? ` ${className}` : ''}`}
			{...delegated}
			ref={buttonRef}
			style={{ '--scale-animation-duration': `${halfAnimation / 1.5}ms` }}
			loading={isLoading}
		>
			{children}
		</SlButton>
	);
});

export default ConfirmationButton;
