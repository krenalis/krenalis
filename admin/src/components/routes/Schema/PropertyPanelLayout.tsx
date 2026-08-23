import React, { ReactNode } from 'react';
import './PropertyPanelLayout.css';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';

interface PropertyPanelLayoutProps {
	actions?: ReactNode;
	children: ReactNode;
	className?: string;
	closeLabel?: string;
	onClose?: () => void;
	title?: string;
	titleAdornment?: ReactNode;
}

const PropertyPanelLayout = ({
	actions,
	children,
	className,
	closeLabel,
	onClose,
	title = 'Property',
	titleAdornment,
}: PropertyPanelLayoutProps) => {
	return (
		<aside className={`property-panel${className == null ? '' : ` ${className}`}`}>
			<div className='property-panel__header'>
				<div className='property-panel__title-row'>
					<div className='property-panel__title'>{title}</div>
					{titleAdornment}
				</div>
				<div className='property-panel__header-actions'>
					{actions}
					{onClose != null && <SlIconButton name='x-lg' label={closeLabel} onClick={onClose} />}
				</div>
			</div>
			<div className='property-panel__body'>{children}</div>
		</aside>
	);
};

export { PropertyPanelLayout };
