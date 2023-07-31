import './LittleLogo.css';

const LittleLogo = ({ icon }) => {
	let logo;
	if (icon === '') {
		logo = `<div class='unknownLogo'>?</div>`;
	} else {
		logo = icon;
	}
	return <div className='littleLogo' dangerouslySetInnerHTML={{ __html: logo }}></div>;
};

export default LittleLogo;
