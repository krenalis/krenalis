import React from 'react';
import './SnowflakeSettings.css';
import { WarehouseSettings } from '../../../lib/api/types/warehouse';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';

interface settingsProps {
	setSettings: React.Dispatch<React.SetStateAction<any>>;
	settings: WarehouseSettings | undefined;
}

const SnowflakeSettings = ({ setSettings, settings }: settingsProps) => {
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
				placeholder=''
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.username || ''}
			/>
			<SlInput
				name='password'
				label='Password'
				placeholder=''
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
				placeholder=''
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.database || ''}
			/>
			<SlInput
				name='schema'
				label='Schema'
				placeholder=''
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.schema || ''}
			/>
			<SlInput
				name='warehouse'
				label='Warehouse'
				placeholder=''
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.warehouse || ''}
			/>
			<SlInput
				name='role'
				label='Role'
				placeholder=''
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
