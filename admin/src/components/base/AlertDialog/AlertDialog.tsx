import React, { ReactNode, useContext } from 'react';
import AppContext from '../../../context/AppContext';
import './AlertDialog.css';
import WaDialog from '@awesome.me/webawesome/dist/react/dialog/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';

interface AlertDialogProps {
	isOpen: boolean;
	onClose: () => void;
	onRequestClose?: (event: Event) => void;
	title: ReactNode;
	actions?: ReactNode;
	children?: ReactNode;
	className?: string;
	variant?: string;
}

const AlertDialog = ({
	isOpen,
	onClose,
	onRequestClose,
	title,
	actions,
	children,
	className,
	variant,
}: AlertDialogProps) => {
	const { isFullscreen } = useContext(AppContext);

	let icon: ReactNode, color: string;
	switch (variant) {
		case 'danger':
			icon = <WaIcon name='exclamation-circle-fill'></WaIcon>;
			color = 'var(--wa-color-danger-40)';
			break;
		default:
			color = 'var(--wa-color-neutral-40)';
			break;
	}

	// The dialog emits wa-hide also when its owner closes it. While isOpen is
	// still true, the hide has been requested by the user.
	const onHide = (event: Event) => {
		if (event.target !== event.currentTarget || !isOpen) {
			return;
		}
		onRequestClose?.(event);
	};

	const onAfterHide = (event: Event) => {
		if (event.target !== event.currentTarget) {
			return;
		}
		onClose();
	};

	return (
		<WaDialog
			className={`alert-dialog${className ? ' ' + className : ''}${isFullscreen ? ' alert-dialog--fullscreen' : ''}`}
			open={isOpen}
			lightDismiss
			onWaHide={onHide}
			onWaAfterHide={onAfterHide}
			style={{ '--alert-color': color, '--width': '600px' } as React.CSSProperties}
		>
			<div className='alert-dialog__icon'>{icon}</div>
			<div className='alert-dialog__title'>{title}</div>
			<div className='alert-dialog__content'>{children}</div>
			<div className='alert-dialog__pipelines'>{actions}</div>
		</WaDialog>
	);
};

export default AlertDialog;
