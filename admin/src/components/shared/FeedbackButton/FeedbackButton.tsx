import React, { forwardRef, useImperativeHandle, useRef, useState, ReactNode } from 'react';
import './FeedbackButton.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlButtonType from '@shoelace-style/shoelace/dist/components/button/button.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Size, Variant } from '../../../types/internal/app';

interface FeedbackButtonProps {
	children: ReactNode;
	loading?: boolean;
	className?: string;
	size?: Size;
	animationDuration?: number;
	variant?: Variant;
	onClick?: () => void;
}

interface FeedbackButtonRef {
	confirm: () => void;
	error: (errorContent: ReactNode) => void;
	load: () => void;
	stop: (cb?: (...args: any) => any) => void;
}

const FeedbackButton = forwardRef<FeedbackButtonRef, FeedbackButtonProps>(
	({ children, loading, className, animationDuration, variant, onClick, size = 'medium', ...delegated }, ref) => {
		const [isLoading, setIsLoading] = useState<boolean>(false);
		const [error, setError] = useState<ReactNode>(null);

		const buttonRef = useRef<SlButtonType>(null);
		const loadingDurationRef = useRef<number>(0);

		const duration = animationDuration || 1200;
		const halfAnimation = duration / 2;

		const isAnimating = useRef<boolean>(false);

		const giveFeedback = (feedback: 'confirmation' | 'error', errorContent?: ReactNode) => {
			if (isAnimating.current) {
				return;
			}
			isAnimating.current = true;
			const button = buttonRef.current!;
			const initialContent = button.innerHTML;
			const initialWidth = button.getBoundingClientRect().width;
			let className: string;
			let iconHTML: string;
			if (feedback === 'confirmation') {
				className = 'confirm';
				iconHTML = `<sl-icon name='check' />`;
			} else {
				className = 'error';
				iconHTML = `<sl-icon name='x' />`;
			}
			button.classList.add(className);
			button.style.width = `${initialWidth}px`;
			button.innerHTML = iconHTML;
			setTimeout(() => {
				button.classList.remove(className);
				if (feedback === 'error') {
					setError(errorContent);
				}
				setTimeout(() => {
					button.innerHTML = initialContent!;
					button.style.width = `unset`;
					isAnimating.current = false;
				}, halfAnimation);
			}, halfAnimation);
		};

		const startLoading = () => {
			setError(null);
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
					stopLoading(() => giveFeedback('confirmation'));
				},
				error(errorContent: ReactNode) {
					stopLoading(() => giveFeedback('error', errorContent));
				},
				load() {
					startLoading();
				},
				stop(cb?: (...args: any) => any) {
					stopLoading(cb);
				},
			}),
			[],
		);

		const button = (
			<SlButton
				className={`feedbackButton${className != null ? ` ${className}` : ''}`}
				ref={buttonRef}
				size={size}
				onClick={onClick}
				variant={variant ? variant : 'default'}
				style={{ '--scale-animation-duration': `${halfAnimation / 1.5}ms` } as React.CSSProperties}
				disabled={loading || isLoading}
				loading={loading || isLoading}
				{...delegated}
			>
				{children}
			</SlButton>
		);

		return (
			<SlTooltip
				className='feedbackTooltip'
				open={error !== null ? true : false}
				trigger='manual'
				style={{ '--max-width': '250px' } as React.CSSProperties}
			>
				{error !== null && (
					<div slot='content' className='tooltipContent'>
						<SlIcon className='closeIcon' name='x' onClick={() => setError(null)} />
						<SlIcon className='errorIcon' name='exclamation-circle-fill' />
						{error}
					</div>
				)}
				{button}
			</SlTooltip>
		);
	},
);

export default FeedbackButton;
export { FeedbackButtonRef };
