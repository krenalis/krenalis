import React, { ReactNode } from 'react';
import './Accordion.css';

interface AccordionProps {
	className?: string;
	isOpen: boolean;
	summary: ReactNode;
	details: ReactNode;
}

const Accordion = ({ className, isOpen, summary, details }: AccordionProps) => {
	return (
		<div className={`accordion${isOpen ? ' open' : ''}${className ? ' ' + className : ''}`}>
			<div className='accordion__summary'>{summary}</div>
			<div className='accordion__details'>{details}</div>
		</div>
	);
};

export default Accordion;
