import React, { ReactNode } from 'react';
import UnknownLogo from '../shared/UnknownLogo/UnknownLogo';
import LittleLogo from '../shared/LittleLogo/LittleLogo';

const getConnectorLogo = (connectorIcon: string) => {
	let logo: ReactNode;
	if (connectorIcon === '') {
		logo = <UnknownLogo size={21} />;
	} else {
		logo = <LittleLogo icon={connectorIcon} />;
	}
	return logo;
};

export default getConnectorLogo;
