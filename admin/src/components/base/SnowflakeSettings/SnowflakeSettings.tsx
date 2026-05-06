import React, { useEffect } from 'react';
import './SnowflakeSettings.css';
import { WarehouseSettings } from '../../../lib/api/types/warehouse';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';

interface settingsProps {
	setSettings: React.Dispatch<React.SetStateAction<any>>;
	settings: WarehouseSettings | undefined;
	precompileDefault: boolean;
	inputRef?: any;
}

const SnowflakeSettings = ({ setSettings, settings, precompileDefault, inputRef }: settingsProps) => {
	useEffect(() => {
		if (settings === undefined && precompileDefault) {
			// Precompile schema.
			setSettings({
				schema: 'PUBLIC',
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
				label='Account Identifier'
				placeholder='ABCDEFG-TUVWXYZ'
				minlength={3}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.account || ''}
			/>
			<SlInput
				name='username'
				label='User Name'
				placeholder='USERNAME'
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.username || ''}
				ref={inputRef}
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
				name='role'
				label='Role'
				placeholder='CUSTOM_ROLE'
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.role || ''}
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
				placeholder='PUBLIC'
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.schema || ''}
			/>
			<SlInput
				name='warehouse'
				label='Warehouse'
				placeholder='COMPUTE_WH'
				type='text'
				minlength={1}
				maxlength={255}
				onSlInput={onSettingInput}
				value={settings?.warehouse || ''}
			/>
		</>
	);
};

export { SnowflakeSettings };
