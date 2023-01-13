import './LittleLogo.css';

const LittleLogo = ({ url, alternativeText }) => {
	return (
		<div className='LittleLogo'>
			<img src={url} rel='noreferrer' alt={alternativeText} />
		</div>
	);
};

export default LittleLogo;
