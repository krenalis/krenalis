import { forwardRef } from 'react';
import Flex from '../Flex/Flex';
import './Section.css';

const Section = forwardRef(({ title, description, actions, children, padded }, ref) => {
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
});

export default Section;
