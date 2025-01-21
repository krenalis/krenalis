import React, { ReactNode, useEffect, useLayoutEffect, useRef, useState } from 'react';
import './Accordion.css';

interface AccordionProps {
	className?: string;
	isOpen: boolean;
	summary: ReactNode;
	details: ReactNode;
}

const Accordion = ({ className, isOpen, summary, details }: AccordionProps) => {
	const [isAccordionOpen, setIsAccordionOpen] = useState<boolean>(true);
	const [isRendered, setIsRendered] = useState<boolean>(false);

	const accordionRef = useRef<any>();
	const maxHeightRef = useRef<number>();

	useLayoutEffect(() => {
		const details = accordionRef.current.querySelector('.accordion__details');
		const height = details.clientHeight;
		setTimeout(() => {
			maxHeightRef.current = height;
			setIsAccordionOpen(isOpen);
			setIsRendered(true);
		});
	}, []);

	useEffect(() => {
		if (maxHeightRef.current === null) {
			return;
		}
		setIsAccordionOpen(isOpen);
	}, [isOpen]);

	return (
		<div
			ref={accordionRef}
			className={`accordion${isAccordionOpen ? ' accordion--open' : ''}${className ? ' ' + className : ''}`}
			style={{ visibility: isRendered ? 'visible' : 'hidden' }}
		>
			<div className='accordion__summary'>{summary}</div>
			<div
				className='accordion__details'
				style={{ maxHeight: isAccordionOpen ? `${maxHeightRef.current}px` : 0 }}
			>
				{details}
			</div>
		</div>
	);
};

export default Accordion;
