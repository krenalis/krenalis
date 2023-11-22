import React, { useContext, useState, useLayoutEffect } from 'react';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import LittleLogo from '../../shared/LittleLogo/LittleLogo';
import { Warehouse } from './DataWarehouse.helpers';
import appContext from '../../../context/AppContext';
import * as icons from '../../../constants/icons';
import * as variants from '../../../constants/variants';
import { WarehouseSettings } from '../../../types/external/warehouse';
import objectKeysToLower from '../../../lib/utils/objectKeysToLower';

interface DataWarehouseSettingsProps {
	selectedWarehouse: Warehouse;
	setSelectedWarehouse: React.Dispatch<React.SetStateAction<Warehouse | undefined>>;
	currentSettings: WarehouseSettings | undefined;
}

const DataWarehouseSettings = ({
	selectedWarehouse,
	setSelectedWarehouse,
	currentSettings,
}: DataWarehouseSettingsProps) => {
	const [settings, setSettings] = useState<Record<string, any> | undefined>(objectKeysToLower(currentSettings));
	const [isPingLoading, setIsPingLoading] = useState<boolean>(false);
	const [isActionButtonLoading, setIsActionButtonLoading] = useState<boolean>(false);

	const { setTitle, api, showError, showStatus, setIsLoadingState } = useContext(appContext);

	useLayoutEffect(() => {
		setTitle(`${selectedWarehouse.label} settings`);
	}, []);

	const onCancelClick = () => setSelectedWarehouse(null);

	const onPing = async () => {
		const timeout = setTimeout(() => setIsPingLoading(true), 300);
		try {
			await api.workspaces.pingWarehouse(selectedWarehouse.label, settings);
		} catch (err) {
			showError(err);
			clearTimeout(timeout);
			setIsPingLoading(false);
			return;
		}
		showStatus({
			variant: variants.SUCCESS,
			icon: icons.OK,
			text: `${selectedWarehouse.label} responded succesfully`,
		});
		clearTimeout(timeout);
		setIsPingLoading(false);
	};

	const onConnect = async () => {
		setIsActionButtonLoading(true);
		try {
			await api.workspaces.connectWarehouse(selectedWarehouse.label, settings);
		} catch (err) {
			showError(err);
			setIsActionButtonLoading(false);
			return;
		}
		setTimeout(() => {
			setIsActionButtonLoading(false);
			setSelectedWarehouse(null);
			setIsLoadingState(true);
		}, 500);
	};

	const onSave = async () => {
		setIsActionButtonLoading(true);
		try {
			await api.workspaces.changeWarehouseSettings(selectedWarehouse.label, settings);
		} catch (err) {
			showError(err);
			setIsActionButtonLoading(false);
			return;
		}
		setTimeout(() => {
			setIsActionButtonLoading(false);
			setSelectedWarehouse(null);
		}, 500);
	};

	return (
		<div className='warehouse-settings'>
			<div className='warehouse-settings__info'>
				<div className='warehouse-settings__icon'>
					<LittleLogo icon={selectedWarehouse.icon} />
				</div>
				<p className='warehouse-settings__name'>{selectedWarehouse.label}</p>
			</div>
			<div className='warehouse-settings__settings'>
				{selectedWarehouse.name === 'postgresql' ? (
					<PostgreSQLSettings setSettings={setSettings} settings={settings} />
				) : selectedWarehouse.name === 'clickhouse' ? (
					<ClickhouseSettings setSettings={setSettings} settings={settings} />
				) : (
					<SnowflakeSettings setSettings={setSettings} settings={settings} />
				)}
			</div>

			<div className='warehouse-settings__buttons'>
				<SlButton disabled={isPingLoading || isActionButtonLoading} variant='default' onClick={onCancelClick}>
					Cancel
				</SlButton>
				<SlButton
					disabled={isPingLoading || isActionButtonLoading}
					loading={isPingLoading}
					variant='default'
					onClick={onPing}
				>
					Ping
				</SlButton>
				<SlButton
					disabled={isPingLoading || isActionButtonLoading}
					loading={isActionButtonLoading}
					variant='primary'
					onClick={currentSettings ? onSave : onConnect}
				>
					{currentSettings ? 'Save' : 'Connect'}
				</SlButton>
			</div>
		</div>
	);
};

interface settingsProps {
	setSettings: React.Dispatch<React.SetStateAction<any>>;
	settings: WarehouseSettings | undefined;
}

const PostgreSQLSettings = ({ setSettings, settings }: settingsProps) => {
	const onSettingChange = (e) => {
		const name = e.currentTarget.name;
		let value = e.currentTarget.value;
		if (name === 'port') {
			value = Number(value);
		}
		setSettings((prevSettings: any) => {
			return {
				...prevSettings,
				[name]: value,
			};
		});
	};

	return (
		<>
			<SlInput
				name='host'
				label='Host'
				placeholder='example.com'
				minlength={1}
				maxlength={253}
				onSlChange={onSettingChange}
				value={settings?.host || ''}
			/>
			<SlInput
				name='port'
				label='Port'
				placeholder='5432'
				type='number'
				minlength={1}
				maxlength={5}
				onSlChange={onSettingChange}
				value={settings?.port || ''}
			/>
			<SlInput
				name='username'
				label='Username'
				placeholder='username'
				type='text'
				minlength={1}
				maxlength={63}
				onSlChange={onSettingChange}
				value={settings?.username || ''}
			/>
			<SlInput
				name='password'
				label='Password'
				placeholder='password'
				type='password'
				minlength={1}
				maxlength={100}
				onSlChange={onSettingChange}
				value={settings?.password || ''}
				password-toggle
			/>
			<SlInput
				name='database'
				label='Database name'
				placeholder='database'
				type='text'
				minlength={1}
				maxlength={63}
				onSlChange={onSettingChange}
				value={settings?.database || ''}
			/>
			<SlInput
				name='schema'
				label='Schema'
				placeholder='public'
				type='text'
				minlength={1}
				maxlength={63}
				onSlChange={onSettingChange}
				value={settings?.schema || ''}
			/>
		</>
	);
};

const ClickhouseSettings = ({ setSettings, settings }: settingsProps) => {
	const onSettingChange = (e) => {
		const name = e.currentTarget.name;
		let value = e.currentTarget.value;
		if (name === 'port') {
			value = Number(value);
		}
		setSettings((prevSettings: any) => {
			return {
				...prevSettings,
				[name]: value,
			};
		});
	};

	return (
		<>
			<SlInput
				name='host'
				label='Host'
				placeholder='example.com'
				minlength={1}
				maxlength={253}
				onSlChange={onSettingChange}
				value={settings?.host || ''}
			/>
			<SlInput
				name='port'
				label='Port'
				placeholder='9000'
				type='number'
				minlength={1}
				maxlength={5}
				onSlChange={onSettingChange}
				value={settings?.port || ''}
			/>
			<SlInput
				name='username'
				label='Username'
				placeholder='username'
				type='text'
				minlength={1}
				maxlength={64}
				onSlChange={onSettingChange}
				value={settings?.username || ''}
			/>
			<SlInput
				name='password'
				label='Password'
				placeholder='password'
				type='password'
				minlength={1}
				maxlength={100}
				onSlChange={onSettingChange}
				value={settings?.password || ''}
				password-toggle
			/>
			<SlInput
				name='database'
				label='Database name'
				placeholder='database'
				type='text'
				minlength={1}
				maxlength={64}
				onSlChange={onSettingChange}
				value={settings?.database || ''}
			/>
		</>
	);
};

const SnowflakeSettings = ({ setSettings, settings }: settingsProps) => {
	const onSettingChange = (e) => {
		const name = e.currentTarget.name;
		const value = e.currentTarget.value;
		setSettings((prevSettings: any) => {
			return {
				...prevSettings,
				[name]: value,
			};
		});
	};

	return (
		<>
			<SlInput
				name='account'
				label='Account'
				placeholder='ABCDEFG-TUVWXYZ'
				minlength={1}
				maxlength={255}
				onSlChange={onSettingChange}
				value={settings?.account || ''}
			/>
			<SlInput
				name='username'
				label='Username'
				placeholder=''
				type='text'
				minlength={1}
				maxlength={255}
				onSlChange={onSettingChange}
				value={settings?.username || ''}
			/>
			<SlInput
				name='password'
				label='Password'
				placeholder=''
				type='password'
				minlength={1}
				maxlength={255}
				onSlChange={onSettingChange}
				value={settings?.password || ''}
				password-toggle
			/>
			<SlInput
				name='database'
				label='Database'
				placeholder=''
				type='text'
				minlength={1}
				maxlength={255}
				onSlChange={onSettingChange}
				value={settings?.database || ''}
			/>
			<SlInput
				name='schema'
				label='Schema'
				placeholder=''
				type='text'
				minlength={1}
				maxlength={255}
				onSlChange={onSettingChange}
				value={settings?.schema || ''}
			/>
			<SlInput
				name='warehouse'
				label='Warehouse'
				placeholder=''
				type='text'
				minlength={1}
				maxlength={255}
				onSlChange={onSettingChange}
				value={settings?.warehouse || ''}
			/>
			<SlInput
				name='role'
				label='Role'
				placeholder=''
				type='text'
				minlength={1}
				maxlength={255}
				onSlChange={onSettingChange}
				value={settings?.role || ''}
			/>
		</>
	);
};

export default DataWarehouseSettings;
