import './Header.css';
import IconWrapper from '../../shared/IconWrapper/IconWrapper';
import { SlAvatar } from '@shoelace-style/shoelace/dist/react/index.js';

const Header = ({ title }) => {
	return (
		<div className='header' justifyContent='space-between' alignItems='center'>
			<div className='title'>
				<span>{title}</span>
			</div>
			<div className='account'>
				<IconWrapper name='bell' moat={true}></IconWrapper>
				<IconWrapper name='question-lg' moat={true}></IconWrapper>
				<SlAvatar
					className='accountAvatar'
					image='data:image/jpeg;base64,/9j/'
				/>
			</div>
		</div>
	);
};

export default Header;
