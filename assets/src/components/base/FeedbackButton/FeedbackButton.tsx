import React, { forwardRef, useImperativeHandle, useRef, useState, ReactNode } from 'react';
import './FeedbackButton.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlButtonType from '@shoelace-style/shoelace/dist/components/button/button.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Size, Variant } from '../../routes/App/App.types';

interface FeedbackButtonProps {
	children: ReactNode;
	loading?: boolean;
	className?: string;
	size?: Size;
	hoist?: boolean;
	animationDuration?: number;
	variant?: Variant;
	onClick?: () => void;
	disabled?: boolean;
}

interface FeedbackButtonRef {
	confirm: () => void;
	error: (message: ReactNode) => void;
	info: (message: ReactNode) => void;
	load: () => void;
	stop: (cb?: (...args: any) => any) => void;
	hideTooltip: () => void;
}

const FeedbackButton = forwardRef<FeedbackButtonRef, FeedbackButtonProps>(
	(
		{
			children,
			loading,
			className,
			animationDuration,
			variant,
			onClick,
			size = 'medium',
			hoist = false,
			disabled,
			...delegated
		},
		ref,
	) => {
		const [isLoading, setIsLoading] = useState<boolean>(false);
		const [error, setError] = useState<ReactNode>(null);
		const [info, setInfo] = useState<ReactNode>(null);

		const buttonRef = useRef<SlButtonType>(null);
		const loadingDurationRef = useRef<number>(0);

		const duration = animationDuration || 1200;
		const halfAnimation = duration / 2;

		const isAnimating = useRef<boolean>(false);

		const giveFeedback = (feedback: 'confirmation' | 'error' | 'info', message?: ReactNode) => {
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
				className = 'feedback-button--confirm';
				iconHTML = `<sl-icon name='check' />`;
				button.classList.add(className);
				button.style.width = `${initialWidth}px`;
				button.innerHTML = iconHTML;
				setTimeout(() => {
					button.classList.remove(className);
					setTimeout(() => {
						button.innerHTML = initialContent!;
						button.style.width = `unset`;
						isAnimating.current = false;
					}, halfAnimation);
				}, halfAnimation);
			} else if (feedback === 'error') {
				setError(message);
				isAnimating.current = false;
			} else if (feedback === 'info') {
				setInfo(message);
				isAnimating.current = false;
			}
		};

		const startLoading = () => {
			setError(null);
			setInfo(null);
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
				error(message: ReactNode) {
					stopLoading(() => giveFeedback('error', message));
				},
				info(message: ReactNode) {
					stopLoading(() => giveFeedback('info', message));
				},
				load() {
					startLoading();
				},
				stop(cb?: (...args: any) => any) {
					stopLoading(cb);
				},
				hideTooltip() {
					setError(null);
					setInfo(null);
				},
			}),
			[],
		);

		const button = (
			<SlButton
				className={`feedback-button${className != null ? ` ${className}` : ''}`}
				ref={buttonRef}
				size={size}
				onClick={onClick}
				variant={variant ? variant : 'default'}
				style={{ '--scale-animation-duration': `${halfAnimation / 1.5}ms` } as React.CSSProperties}
				disabled={disabled || loading || isLoading}
				loading={loading || isLoading}
				{...delegated}
			>
				{children}
			</SlButton>
		);

		return (
			<SlTooltip
				className='feedback-button__tooltip'
				open={error !== null || info !== null ? true : false}
				trigger='manual'
				style={{ '--max-width': '400px' } as React.CSSProperties}
				placement='bottom'
				hoist={hoist}
			>
				{(error !== null || info !== null) && (
					<div
						slot='content'
						className={`feedback-button__tooltip-content ${error !== null ? 'feedback-button__tooltip-content--error' : 'feedback-button__tooltip-content--info'}`}
					>
						<SlIcon
							className='feedback-button__tooltip-icon-close'
							name='x-lg'
							onClick={() => {
								if (error !== null) {
									setError(null);
								} else {
									setInfo(null);
								}
							}}
						/>
						{error !== null ? (
							<SlIcon className='feedback-button__tooltip-icon-error' name='exclamation-circle' />
						) : (
							<SlIcon className='feedback-button__tooltip-icon-info' name='info-circle' />
						)}
						{error !== null ? (
							<div className='feedback-button__tooltip-error'>{error}</div>
						) : (
							<div className='feedback-button__tooltip-info'>{info}</div>
						)}
					</div>
				)}
				{button}
			</SlTooltip>
		);
	},
);

export default FeedbackButton;
export { FeedbackButtonRef };
