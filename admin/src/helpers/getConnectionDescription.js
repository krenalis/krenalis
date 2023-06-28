const getConnectionDescription = (connection) => {
	let description;
	if (connection.isSource) {
		description = connection.connector.sourceDescription;
	} else {
		description = connection.connector.destinationDescription;
	}
	return description;
};

export default getConnectionDescription;
