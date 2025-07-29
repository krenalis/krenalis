import React, { useEffect } from 'react';
import './SnowflakeSettings.css';
import { WarehouseSettings } from '../../../lib/api/types/warehouse';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';

interface settingsProps {
	setSettings: React.Dispatch<React.SetStateAction<any>>;
	settings: WarehouseSettings | undefined;
	precompileDefault: boolean;
}

const SnowflakeSettings = ({ setSettings, settings, precompileDefault }: settingsProps) => {
	useEffect(() => {
		if (settings === undefined && precompileDefault) {
			// Precompile schema and role.
			setSettings({
				schema: 'public',
				role: 'SYSADMIN',
			});
		}
	}, []);

	const onSettingInput = (e) => {
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
				placeholder='ABCDEFG.TUVWXYZ'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.account || ''}
			/>
			<SlInput
				name='username'
				label='Username'
				placeholder='username'
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.username || ''}
			/>
			<SlInput
				name='password'
				label='Password'
				placeholder='password'
				type='password'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.password || ''}
				password-toggle
			/>
			<SlInput
				name='database'
				label='Database'
				placeholder='MY_DATABASE'
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.database || ''}
			/>
			<SlInput
				name='schema'
				label='Schema'
				placeholder='public'
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.schema || ''}
			/>
			<SlInput
				name='warehouse'
				label='Warehouse'
				placeholder='MY_WAREHOUSE'
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.warehouse || ''}
			/>
			<SlInput
				name='role'
				label='Role'
				placeholder='SYSADMIN'
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.role || ''}
			/>
		</>
	);
};

export { SnowflakeSettings };
