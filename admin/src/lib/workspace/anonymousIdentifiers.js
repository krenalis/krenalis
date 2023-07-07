const transformAnonymousIdentifiers = (identifiers) => {
	const transformed = [];
	if (identifiers.Priority.length === 0) {
		transformed.push(['', '']);
	} else {
		for (const property of identifiers.Priority) {
			const mapped = identifiers.Mapping[property];
			transformed.push([mapped, property]);
		}
	}
	return transformed;
};

const untransformAnonymousIdentifiers = (transformed) => {
	const untransformed = { Priority: [], Mapping: {} };
	for (const [mapped, property] of transformed) {
		untransformed.Priority.push(property);
		untransformed.Mapping[property] = mapped;
	}
	return untransformed;
};

export { transformAnonymousIdentifiers, untransformAnonymousIdentifiers };
