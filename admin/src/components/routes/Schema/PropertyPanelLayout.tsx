import React, { ReactNode, useState } from 'react';
import './PropertyPanelLayout.css';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';

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
			{onClose != null && (
				<WaButton className='icon-button' appearance='plain' onClick={onClose}>
					<WaIcon name='x-lg' label={closeLabel} />
				</WaButton>
			)}
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
