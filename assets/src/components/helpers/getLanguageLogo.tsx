import React, { ReactNode } from 'react';
import { SQL_LOGO, JS_LOGO, PYTHON_LOGO } from '../../constants/languageLogos';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';

const getLanguageLogo = (language: string): ReactNode => {
	let languageLogo: ReactNode;
	switch (language) {
		case 'SQL':
			languageLogo = <img src={SQL_LOGO} alt='SQL logo' />;
			break;
		case 'JavaScript':
			languageLogo = <img src={JS_LOGO} alt='JavaScript logo' />;
			break;
		case 'Python':
			languageLogo = <img src={PYTHON_LOGO} alt='Python logo' />;
			break;
		default:
			languageLogo = <SlIcon name='file-earmark-code' />;
			break;
	}
	return languageLogo;
};

export default getLanguageLogo;
