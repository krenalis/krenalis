import React, { useEffect } from 'react';
import './SnowflakeSettings.css';
import { WarehouseSettings } from '../../../lib/api/types/warehouse';
import WaInput from '@awesome.me/webawesome/dist/react/input/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';

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
			<WaInput
				name='account'
				label='Account Identifier'
				placeholder='ABCDEFG-TUVWXYZ'
				minlength={3}
				maxlength={255}
				onInput={onSettingInput}
				value={settings?.account || ''}
			/>
			<WaInput
				name='username'
				label='User Name'
				placeholder='USERNAME'
				type='text'
				minlength={1}
				maxlength={255}
				onInput={onSettingInput}
				value={settings?.username || ''}
				ref={inputRef}
			/>
			<WaInput
				name='password'
				label='Password'
				placeholder='password'
				type='password'
				minlength={1}
				maxlength={255}
				onInput={onSettingInput}
				value={settings?.password || ''}
				passwordToggle
			>
				<WaIcon name='eye' slot='show-password-icon' />
				<WaIcon name='eye-slash' slot='hide-password-icon' />
			</WaInput>
			<WaInput
				name='role'
				label='Role'
				placeholder='CUSTOM_ROLE'
				type='text'
				minlength={1}
				maxlength={255}
				onInput={onSettingInput}
				value={settings?.role || ''}
			/>
			<WaInput
				name='database'
				label='Database'
				placeholder='MY_DATABASE'
				type='text'
				minlength={1}
				maxlength={255}
				onInput={onSettingInput}
				value={settings?.database || ''}
			/>
			<WaInput
				name='schema'
				label='Schema'
				placeholder='PUBLIC'
				type='text'
				minlength={1}
				maxlength={255}
				onInput={onSettingInput}
				value={settings?.schema || ''}
			/>
			<WaInput
				name='warehouse'
				label='Warehouse'
				placeholder='COMPUTE_WH'
				type='text'
				minlength={1}
				maxlength={255}
				onInput={onSettingInput}
				value={settings?.warehouse || ''}
			/>
		</>
	);
};

export { SnowflakeSettings };
