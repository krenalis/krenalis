import { forwardRef } from 'react';
import './Toast.css';
import { SlAlert, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const Toast = forwardRef(({ status }, ref) => {
	if (status == null) {
		return <SlAlert ref={ref} variant='neutral' closable></SlAlert>;
	}

	return (
		<SlAlert ref={ref} variant={status.variant} closable>
			<SlIcon slot='icon' name={status.icon} />
			<b>{status.text}</b>
		</SlAlert>
	);
});

export default Toast;
