import React, { forwardRef, ReactNode } from 'react';
import Flex from '../Flex/Flex';
import './Section.css';

interface SectionProps {
	title: string;
	description?: string;
	actions?: ReactNode[] | ReactNode;
	children: ReactNode;
	padded?: boolean;
	className?: string;
}

const Section = forwardRef<any, SectionProps>(
	({ title, description, actions, children, padded, className }: SectionProps, ref) => {
		return (
			<div className={`section${className ? ' ' + className : ''}`} ref={ref}>
				<Flex justifyContent='space-between' alignItems='center'>
					<div className='section__text'>
						<div className='section__title'>{title}</div>
						{description && <div className='section__description'>{description}</div>}
					</div>
					<Flex className='section__actions' justifyContent='end' alignItems='center'>
						{actions}
					</Flex>
				</Flex>
				<div className={`section__content${padded ? ' section__content--padded' : ''}`}>{children}</div>
			</div>
		);
	},
);

export default Section;
