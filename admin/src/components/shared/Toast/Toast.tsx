import React, { forwardRef } from 'react';
import './Toast.css';
import SlAlert from '@shoelace-style/shoelace/dist/react/alert/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Status } from '../../../types/internal/app';

interface ToastProps {
	status: Status | null;
	isFullscreen: boolean;
}

const Toast = forwardRef<any, ToastProps>(({ status, isFullscreen }, ref) => {
	if (status == null) {
		return <SlAlert ref={ref} variant='neutral' closable></SlAlert>;
	}

	return (
		<SlAlert className={isFullscreen ? 'isFullscreen' : ''} ref={ref} variant={status.variant} closable>
			<SlIcon slot='icon' name={status.icon} />
			<b>{status.text}</b>
		</SlAlert>
	);
});

export default Toast;
