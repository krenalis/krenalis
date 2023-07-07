import { useMemo } from 'react';
import { transformAnonymousIdentifiers, untransformAnonymousIdentifiers } from '../lib/workspace/anonymousIdentifiers';

const useTransformedAnonymousIdentifiers = (identifiers, setIdentifiers) => {
	const transformedAnonymousIdentifiers = useMemo(() => {
		return transformAnonymousIdentifiers(identifiers);
	}, [identifiers]);

	const setTransformedAnonymousIdentifiers = (identifiers) => {
		const untransformed = untransformAnonymousIdentifiers(identifiers);
		setIdentifiers(untransformed);
	};

	return { transformedAnonymousIdentifiers, setTransformedAnonymousIdentifiers };
};

export default useTransformedAnonymousIdentifiers;
