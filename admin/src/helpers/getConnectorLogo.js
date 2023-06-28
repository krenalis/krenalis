import UnknownLogo from '../components/common/UnknownLogo/UnknownLogo';
import LittleLogo from '../components/common/LittleLogo/LittleLogo';

const getConnectorLogo = (connectorIcon) => {
	let logo;
	if (connectorIcon === '') {
		logo = <UnknownLogo size={21} />;
	} else {
		logo = <LittleLogo icon={connectorIcon} />;
	}
	return logo;
};

export default getConnectorLogo;
