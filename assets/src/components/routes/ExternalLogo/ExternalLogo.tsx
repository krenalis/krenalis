import React, { useContext, useState } from 'react';
import './ExternalLogo.css';
import AppContext from '../../../context/AppContext';
import UnknownLogo from '../../base/UnknownLogo/UnknownLogo';

interface ExternalLogoProps {
	code: string | null;
	slot?: string;
}

const ExternalLogo = ({ code, slot }: ExternalLogoProps) => {
	const [index, setIndex] = useState(0);

	const { publicMetadata } = useContext(AppContext);

	const externalURLs = publicMetadata.externalAssetsURLs;

	if (code == null || index >= externalURLs.length) {
		return <UnknownLogo size={21} />;
	}

	return (
		<img
			src={externalURLs[index] + code + '.svg'}
			onError={() => {
				setIndex(index + 1);
			}}
			slot={slot != null ? slot : undefined}
		/>
	);
};

export { ExternalLogo };
