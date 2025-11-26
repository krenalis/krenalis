import React, { forwardRef, ReactNode } from 'react';
import Flex from '../Flex/Flex';
import './Section.css';

interface SectionProps {
	title: string;
	description?: ReactNode;
	pipelines?: ReactNode[] | ReactNode;
	children: ReactNode;
	padded?: boolean;
	annotated?: boolean;
	className?: string;
}

const Section = forwardRef<any, SectionProps>(
	({ title, description, pipelines, children, padded, annotated, className }: SectionProps, ref) => {
		return (
			<div
				className={`section${className ? ' ' + className : ''}${padded ? ' section--padded' : ''}${annotated ? ' section--annotated' : ''}`}
				ref={ref}
			>
				<div className='section__heading'>
					<div className='section__text'>
						<div className='section__title'>{title}</div>
						{description && <div className='section__description'>{description}</div>}
					</div>
					<Flex className='section__pipelines' justifyContent='end' alignItems='center'>
						{pipelines}
					</Flex>
				</div>
				<div className='section__content'>{children}</div>
			</div>
		);
	},
);

export default Section;
