import React from 'react';
import './PostgreSQLSettings.css';
import { WarehouseSettings } from '../../../lib/api/types/warehouse';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';

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

export { PostgreSQLSettings };
