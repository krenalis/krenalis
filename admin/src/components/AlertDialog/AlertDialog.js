import './AlertDialog.css';
import { SlDialog, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const AlertDialog = ({ variant, isOpen, onClose, title, actions, children }) => {
	let icon, color;
	switch (variant) {
		case 'danger':
			icon = <SlIcon name='exclamation-circle-fill'></SlIcon>;
			color = 'var(--sl-color-danger-600)';
			break;
		default:
			break;
	}

	return (
		<SlDialog
			className={`alertDialog`}
			open={isOpen}
			onSlAfterHide={onClose}
			style={{ '--alert-color': color, '--width': '600px' }}
		>
			<div className='alertIcon'>{icon}</div>
			<div className='alertTitle'>{title}</div>
			<div className='alertContent'>{children}</div>
			<div className='alertActions'>{actions}</div>
		</SlDialog>
	);
};

export default AlertDialog;
