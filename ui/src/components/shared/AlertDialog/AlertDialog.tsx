import React, { ReactNode, useContext } from 'react';
import AppContext from '../../../context/AppContext';
import './AlertDialog.css';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface AlertDialogProps {
	variant: string;
	isOpen: boolean;
	onClose: () => void;
	title: string;
	actions?: ReactNode;
	children?: ReactNode;
}

const AlertDialog = ({ variant, isOpen, onClose, title, actions, children }: AlertDialogProps) => {
	const { isFullscreen } = useContext(AppContext);

	let icon: ReactNode, color: string;
	switch (variant) {
		case 'danger':
			icon = <SlIcon name='exclamation-circle-fill'></SlIcon>;
			color = 'var(--sl-color-danger-600)';
			break;
		default:
			color = 'var(--sl-color-neutral-600)';
			break;
	}

	return (
		<SlDialog
			className={`alertDialog${isFullscreen ? ' fullscreen' : ''}`}
			open={isOpen}
			onSlAfterHide={onClose}
			style={{ '--alert-color': color, '--width': '600px' } as React.CSSProperties}
		>
			<div className='alertIcon'>{icon}</div>
			<div className='alertTitle'>{title}</div>
			<div className='alertContent'>{children}</div>
			<div className='alertActions'>{actions}</div>
		</SlDialog>
	);
};

export default AlertDialog;
