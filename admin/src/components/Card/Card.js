import './Card.css';
import { SlBadge } from '@shoelace-style/shoelace/dist/react/index.js';

const Card = ({ logoURL, name, type, description, children }) => {
	return (
		<div className='Card'>
			<div className='top'>
				<div className='logo'>
					{logoURL === '' ? (
						<div class='unknownLogo'>?</div>
					) : (
						<img alt={`${name}'s logo`} rel='noreferrer' src={logoURL} />
					)}
				</div>
				<div className='name'>{name}</div>
				{type && (
					<SlBadge className='type' variant='neutral'>
						{type}
					</SlBadge>
				)}
				<div className='description'>{description}</div>
			</div>
			<div className='body'>{children}</div>
		</div>
	);
};

export default Card;
