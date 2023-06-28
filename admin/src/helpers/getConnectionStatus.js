import * as variants from '../constants/variants';

const getConnectionStatus = (connection) => {
	if (!connection.enabled) {
		return { text: 'Disabled', variant: variants.NEUTRAL };
	} else {
		switch (connection.health) {
			case 'Healthy':
				return { text: 'Working properly', variant: variants.SUCCESS };
			case 'NoRecentData':
				return { text: 'No recent Data', variant: variants.DANGER };
			case 'RecentError':
				return { text: 'Recent error', variant: variants.DANGER };
			case 'AccessDenied':
				return { text: 'Access denied', variant: variants.DANGER };
			default:
				return { text: null, variant: null };
		}
	}
};

export default getConnectionStatus;
