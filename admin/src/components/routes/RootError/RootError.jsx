import './RootError.css';
import IconWrapper from '../../shared/IconWrapper/IconWrapper';

const RootError = () => {
	return (
		<div className='root-error'>
			<div className='root-error__message'>
				<IconWrapper name='bug' size={40} />
				<div className='root-error__title'>An error occured in the application</div>
				<div className='root-error__instructions'>Please contact the administrator</div>
			</div>
		</div>
	);
};

export default RootError;
