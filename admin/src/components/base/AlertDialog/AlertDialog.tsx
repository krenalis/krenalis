import React, { ReactNode, useContext } from 'react';
import AppContext from '../../../context/AppContext';
import './AlertDialog.css';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

interface AlertDialogProps {
	isOpen: boolean;
	onClose: () => void;
	title: ReactNode;
	actions?: ReactNode;
	children?: ReactNode;
	className?: string;
	variant?: string;
}

const AlertDialog = ({ isOpen, onClose, title, actions, children, className, variant }: AlertDialogProps) => {
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
			className={`alert-dialog${className ? ' ' + className : ''}${isFullscreen ? ' alert-dialog--fullscreen' : ''}`}
			open={isOpen}
			onSlAfterHide={onClose}
			style={{ '--alert-color': color, '--width': '600px' } as React.CSSProperties}
		>
			<div className='alert-dialog__icon'>{icon}</div>
			<div className='alert-dialog__title'>{title}</div>
			<div className='alert-dialog__content'>{children}</div>
			<div className='alert-dialog__actions'>{actions}</div>
		</SlDialog>
	);
};

export default AlertDialog;
