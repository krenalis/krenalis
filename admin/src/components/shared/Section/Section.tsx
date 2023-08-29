import React, { forwardRef, ReactNode } from 'react';
import Flex from '../Flex/Flex';
import './Section.css';

interface SectionProps {
	title: string;
	description?: string;
	actions?: ReactNode[] | ReactNode;
	children: ReactNode;
	padded?: boolean;
}

const Section = forwardRef<any, SectionProps>(
	({ title, description, actions, children, padded }: SectionProps, ref) => {
		return (
			<div className='section' ref={ref}>
				<Flex justifyContent='space-between' alignItems='center'>
					<div className='sectionText'>
						<div className='sectionTitle'>{title}</div>
						{description && <div className='sectionDescription'>{description}</div>}
					</div>
					<Flex className='sectionActions' justifyContent='end' alignItems='center'>
						{actions}
					</Flex>
				</Flex>
				<div className={`sectionContent${padded ? ' padded' : ''}`}>{children}</div>
			</div>
		);
	}
);

export default Section;
