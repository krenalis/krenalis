import './Header.css';
import IconWrapper from '../IconWrapper/IconWrapper';
import { SlAvatar, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { useNavigate } from 'react-router';

const Header = ({ title, previousRoute }) => {
	const navigate = useNavigate();

	const onGoBackIconClick = () => {
		return navigate(previousRoute);
	};

	return (
		<div className='Header' justifyContent='space-between' alignItems='center'>
			<div className='title'>
				{previousRoute !== '' && (
					<SlIcon className='goBackIcon' name='arrow-90deg-up' onClick={onGoBackIconClick}></SlIcon>
				)}
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
