import React, { forwardRef, useImperativeHandle, useRef, useState, ReactNode } from 'react';
import './ConfirmationButton.css';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';
import SlButtonType from '@shoelace-style/shoelace/dist/components/button/button';
import { Size, Variant } from '../../../types/internal/app';

interface ConfirmationButtonProps {
	children: ReactNode;
	className?: string;
	size?: Size;
	animationDuration?: number;
	variant?: Variant;
	onClick?: () => void;
}

interface ConfirmationButtonRef {
	confirm: () => void;
	load: () => void;
	stop: (cb?: (...args: any) => any) => void;
}

const ConfirmationButton = forwardRef<ConfirmationButtonRef, ConfirmationButtonProps>(
	({ children, className, animationDuration, variant = 'default', onClick, size = 'medium', ...delegated }, ref) => {
		const [isLoading, setIsLoading] = useState(false);

		const buttonRef = useRef<SlButtonType>(null);
		const loadingDurationRef = useRef<number>(0);

		const duration = animationDuration || 1200;
		const halfAnimation = duration / 2;

		let isAnimating = false;
		const confirmAnimation = () => {
			if (isAnimating) {
				return;
			}
			isAnimating = true;
			const button = buttonRef.current!;
			const initialText = button.textContent;
			const initialWidth = button.getBoundingClientRect().width;

			button.classList.add('confirm');
			button.style.width = `${initialWidth}px`;

			const icon = `<sl-icon name='check'></sl-icon>`;
			button.innerHTML = icon;

			setTimeout(() => {
				button.classList.remove('confirm');
				setTimeout(() => {
					button.innerHTML = initialText!;
					button.style.width = `unset`;
					isAnimating = false;
				}, halfAnimation);
			}, halfAnimation);
		};

		const startLoading = () => {
			loadingDurationRef.current = Date.now();
			setIsLoading(true);
		};

		const stopLoading = (cb?: (...args: any) => any) => {
			const now = Date.now();
			const delta = now - loadingDurationRef.current;
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
				stop(cb?: (...args: any) => any) {
					stopLoading(cb);
				},
			}),
			[]
		);

		return (
			<SlButton
				className={`confirmationButton${className != null ? ` ${className}` : ''}`}
				ref={buttonRef}
				size={size}
				onClick={onClick}
				variant={variant}
				style={{ '--scale-animation-duration': `${halfAnimation / 1.5}ms` } as React.CSSProperties}
				loading={isLoading}
				{...delegated}
			>
				{children}
			</SlButton>
		);
	}
);

export default ConfirmationButton;
export { ConfirmationButtonRef };
