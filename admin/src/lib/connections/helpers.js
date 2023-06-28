const splitConnectionsByRole = (connections) => {
	const sources = [];
	const destinations = [];
	for (const c of connections) {
		if (c.role === 'Source') sources.push(c);
		if (c.role === 'Destination') destinations.push(c);
	}
	return {
		sources,
		destinations,
	};
};

export { splitConnectionsByRole };
