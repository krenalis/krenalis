import React, { ReactNode, useLayoutEffect, useRef, useState } from 'react';
import './Accordion.css';

interface AccordionProps {
	className?: string;
	isOpen: boolean;
	summary: ReactNode;
	details: ReactNode;
}

const Accordion = ({ className, isOpen, summary, details }: AccordionProps) => {
	const [renderDetails, setRenderDetails] = useState<boolean>();
	const [showDetails, setShowDetails] = useState<boolean>();

	const accordionRef = useRef<any>();
	const maxHeightRef = useRef<number>();
	const isFirstRender = useRef<boolean>(true);

	useLayoutEffect(() => {
		const container = accordionRef.current?.querySelector('.accordion__details');
		if (container == null) {
			return;
		}
		if (isOpen) {
			if (maxHeightRef.current == null) {
				setRenderDetails(true);
			} else {
				setShowDetails(true);
			}
		} else if (!isFirstRender.current) {
			const onEnd = (e: TransitionEvent) => {
				if (e.propertyName === 'max-height') {
					setShowDetails(false);
					container.removeEventListener('transitionend', onEnd);
				}
			};
			container.addEventListener('transitionend', onEnd, { once: true });
		}
		if (isFirstRender.current) {
			isFirstRender.current = false;
		}
	}, [isOpen]);

	useLayoutEffect(() => {
		if (!renderDetails) {
			return;
		}
		maxHeightRef.current = computeDetailsHeight();
		setRenderDetails(false);
		setShowDetails(true);
	}, [renderDetails]);

	const computeDetailsHeight = () => {
		const root = accordionRef.current;
		if (root == null) {
			return 0;
		}

		const container = root.querySelector('.accordion__details');
		if (container == null) {
			return 0;
		}

		const el = document.createElement('div');
		el.style.position = 'absolute';
		el.style.visibility = 'hidden';
		el.style.pointerEvents = 'none';
		el.style.contain = 'layout style';
		el.style.inset = '0 auto auto 0';
		el.style.width = `${root.offsetWidth}px`;
		document.body.appendChild(el);

		const clone = container.cloneNode(true) as HTMLElement;
		clone.style.height = 'auto';
		clone.style.maxHeight = 'none';
		clone.style.overflow = 'visible';

		el.appendChild(clone);

		const height = clone.getBoundingClientRect().height;

		document.body.removeChild(el);
		return height;
	};

	return (
		<div
			ref={accordionRef}
			className={`accordion${isOpen ? ' accordion--open' : ''}${className ? ' ' + className : ''}`}
		>
			<div className='accordion__summary'>{summary}</div>
			<div
				className='accordion__details'
				style={{
					maxHeight: isOpen ? `${maxHeightRef.current}px` : 0,
				}}
			>
				{(renderDetails || showDetails) && details}
			</div>
		</div>
	);
};

export default Accordion;
