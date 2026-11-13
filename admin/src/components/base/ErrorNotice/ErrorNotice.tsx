import React, { ReactNode } from 'react';
import './ErrorNotice.css';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';

interface ErrorNoticeProps {
	title: string;
	description?: ReactNode;
	action?: ReactNode;
	className?: string;
}

const ErrorNotice = ({ title, description, action, className }: ErrorNoticeProps) => {
	const formattedDescription =
		typeof description === 'string'
			? description
					.split(/«([^»]*)»/g)
					.map((part, index) => (index % 2 === 0 ? part : <strong key={index}>{part}</strong>))
			: description;
	return (
		<div className={`error-notice${className != null ? ` ${className}` : ''}`} role='alert'>
			<WaIcon name='exclamation-triangle' aria-hidden='true' />
			<div className='error-notice__content'>
				<div className='error-notice__title'>{title}</div>
				{description != null && <div className='error-notice__description'>{formattedDescription}</div>}
			</div>
			{action != null && <div className='error-notice__action'>{action}</div>}
		</div>
	);
};

export { ErrorNotice };
