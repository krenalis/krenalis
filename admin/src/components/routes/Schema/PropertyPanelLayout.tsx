import React, { ReactNode, useState } from 'react';
import './PropertyPanelLayout.css';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';

interface PropertyPanelLayoutProps {
	actions?: ReactNode;
	actionsAfterContent?: boolean;
	children: ReactNode;
	className?: string;
	closeLabel?: string;
	onClose?: () => void;
	title?: string;
	titleAdornment?: ReactNode;
}

const PropertyPanelLayout = ({
	actions,
	actionsAfterContent = false,
	children,
	className,
	closeLabel,
	onClose,
	title = 'Property',
	titleAdornment,
}: PropertyPanelLayoutProps) => {
	const [isBodyScrolled, setIsBodyScrolled] = useState(false);
	const actionContainer = (actions != null || onClose != null) && (
		<div className='property-panel__header-actions'>
			{actions}
			{onClose != null && <SlIconButton name='x-lg' label={closeLabel} onClick={onClose} />}
		</div>
	);

	return (
		<aside
			className={`property-panel${isBodyScrolled ? ' property-panel--body-scrolled' : ''}${className == null ? '' : ` ${className}`}`}
		>
			<div className='property-panel__header'>
				<div className='property-panel__title-row'>
					<div className='property-panel__title'>{title}</div>
					{titleAdornment}
				</div>
			</div>
			{!actionsAfterContent && actionContainer}
			<div className='property-panel__header-shadow' />
			<div
				className='property-panel__body'
				onScroll={(event) => setIsBodyScrolled(event.currentTarget.scrollTop > 0)}
			>
				{children}
			</div>
			{/* Form actions follow the content in the DOM so they come after its controls in the keyboard focus order. */}
			{actionsAfterContent && actionContainer}
		</aside>
	);
};

export { PropertyPanelLayout };
