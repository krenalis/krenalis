const getStorageFileConnections = (storageID, connections) => {
	return connections.filter((c) => c.storage === storageID);
};

export default getStorageFileConnections;
