import React, { forwardRef, useImperativeHandle, useRef } from 'react';
import './Toast.css';
import WaCallout from '@awesome.me/webawesome/dist/react/callout/index.js';
import WaCalloutElement from '@awesome.me/webawesome/dist/components/callout/callout.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import { Status } from '../../routes/App/App.types';

interface ToastProps {
	status: Status | null;
	isFullscreen: boolean;
}

// ToastRef shows and hides the toast. Both return once the animation has
// finished.
interface ToastRef {
	toast: () => Promise<void>;
	hide: () => Promise<void>;
}

const showKeyframes = [
	{ opacity: 0, scale: 0.8 },
	{ opacity: 1, scale: 1 },
];
const animationOptions = { duration: 250, easing: 'ease' };

// Toast is a manual popover, so that it is drawn in the top layer: above the
// modal dialogs opened before it, which no z-index can reach. While a modal
// dialog is open the rest of the page is inert, the toast included.
const Toast = forwardRef<ToastRef, ToastProps>(({ status, isFullscreen }, ref) => {
	const stackRef = useRef<HTMLDivElement>(null);
	const calloutRef = useRef<WaCalloutElement>(null);

	const animate = async (keyframes: Keyframe[]) => {
		const callout = calloutRef.current;
		callout.getAnimations().forEach((animation) => animation.cancel());
		try {
			await callout.animate(keyframes, animationOptions).finished;
		} catch {
			// The animation has been canceled by a later one.
		}
	};

	const hide = async () => {
		const stack = stackRef.current;
		if (stack == null || !stack.matches(':popover-open')) {
			return;
		}
		await animate([...showKeyframes].reverse());
		stack.hidePopover();
	};

	useImperativeHandle(
		ref,
		() => ({
			async toast() {
				const stack = stackRef.current;
				if (stack.matches(':popover-open')) {
					// Show it again, to put it above the modal dialogs opened since.
					stack.hidePopover();
				}
				stack.showPopover();
				await animate(showKeyframes);
			},
			hide,
		}),
		[],
	);

	return (
		// The React 18 types in use do not know the popover attribute.
		<div className='toast-stack' {...{ popover: 'manual' }} ref={stackRef}>
			<WaCallout
				className={`toast${isFullscreen ? ' toast--fullscreen' : ''}`}
				ref={calloutRef}
				variant={status?.variant === 'primary' ? 'brand' : (status?.variant ?? 'neutral')}
				appearance='accent'
				role='alert'
			>
				{status != null && (
					<>
						<WaIcon slot='icon' name={status.icon} />
						<b className='toast__text'>{status.text}</b>
						<button type='button' className='toast__close' aria-label='Close' onClick={hide}>
							<WaIcon name='x-lg' />
						</button>
					</>
				)}
			</WaCallout>
		</div>
	);
});

export default Toast;
export { ToastRef };
