import React, { ReactNode } from 'react';
import UnknownLogo from '../base/UnknownLogo/UnknownLogo';
import LittleLogo from '../base/LittleLogo/LittleLogo';

const getConnectorLogo = (connectorIcon: string | null) => {
	let logo: ReactNode;
	if (connectorIcon == null || connectorIcon === '') {
		logo = <UnknownLogo size={21} />;
	} else {
		logo = <LittleLogo icon={connectorIcon} />;
	}
	return logo;
};

export default getConnectorLogo;
