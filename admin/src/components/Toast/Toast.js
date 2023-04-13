import React from 'react';
import './Toast.css';
import { SlAlert, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const Toast = ({ toastRef, status }) => {
	if (status == null) {
		return <SlAlert ref={toastRef} variant='neutral' closable></SlAlert>;
	}

	return (
		<SlAlert ref={toastRef} variant={status.variant} closable>
			<SlIcon slot='icon' name={status.icon} />
			<b>{status.text}</b>
		</SlAlert>
	);
};

export default Toast;
