const getConnectionStatusInfos = (connection) => {
	if (!connection.Enabled) {
		return { text: 'Disabled', variant: 'neutral' };
	} else {
		switch (connection.Health) {
			case 'Healthy':
				return { text: 'Working properly', variant: 'success' };
			case 'NoRecentData':
				return { text: 'No recent Data', variant: 'danger' };
			case 'RecentError':
				return { text: 'Recent error', variant: 'danger' };
			case 'AccessDenied':
				return { text: 'Access denied', variant: 'danger' };
			default:
				return { text: null, variant: null };
		}
	}
};

export default getConnectionStatusInfos;
