import Flex from '../Flex/Flex';
import './Section.css';

const Section = ({ title, description, actions, children }) => {
	return (
		<div className='section'>
			<Flex justifyContent='space-between' alignItems='center'>
				<div className='sectionText'>
					<div className='sectionTitle'>{title}</div>
					{description && <div className='sectionDescription'>{description}</div>}
				</div>
				<Flex className='sectionActions' justifyContent='end' alignItems='center'>
					{actions}
				</Flex>
			</Flex>
			<div className='sectionContent'>{children}</div>
		</div>
	);
};

export default Section;
