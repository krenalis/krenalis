import React, { useState } from 'react';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import { sleep } from '../../../utils/sleep';

const MIN_REFRESH_LOADING_DURATION_MS = 400;

interface ProfilesRefreshButtonProps {
	disabled: boolean;
	onRefresh: () => Promise<void>;
}

const ProfilesRefreshButton = ({ disabled, onRefresh }: ProfilesRefreshButtonProps) => {
	const [isRefreshing, setIsRefreshing] = useState(false);
	const refresh = async () => {
		if (disabled || isRefreshing) {
			return;
		}
		setIsRefreshing(true);
		const minimumLoading = sleep(MIN_REFRESH_LOADING_DURATION_MS);
		try {
			await onRefresh();
		} finally {
			await minimumLoading;
			setIsRefreshing(false);
		}
	};

	return (
		<WaButton
			appearance='outlined'
			size='s'
			onClick={refresh}
			disabled={disabled || isRefreshing}
			loading={isRefreshing}
		>
			<WaIcon slot='start' name='arrow-clockwise' />
			Refresh
		</WaButton>
	);
};

export { ProfilesRefreshButton };
