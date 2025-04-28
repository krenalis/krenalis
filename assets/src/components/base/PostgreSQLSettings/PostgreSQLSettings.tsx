import React, { useEffect } from 'react';
import './PostgreSQLSettings.css';
import { WarehouseSettings } from '../../../lib/api/types/warehouse';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';

interface settingsProps {
	setSettings: React.Dispatch<React.SetStateAction<any>>;
	settings: WarehouseSettings | undefined;
}

const PostgreSQLSettings = ({ setSettings, settings }: settingsProps) => {
	useEffect(() => {
		if (settings === undefined) {
			// Precompile port and schema.
			setSettings({
				port: 5432,
				schema: 'public',
			});
		}
	}, []);

	const onSettingInput = (e) => {
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
				onSlInput={onSettingInput}
				value={settings?.host || ''}
			/>
			<SlInput
				name='port'
				label='Port'
				placeholder='5432'
				type='number'
				minlength={1}
				maxlength={5}
				onSlInput={onSettingInput}
				value={settings?.port || ''}
			/>
			<SlInput
				name='username'
				label='Username'
				placeholder='username'
				type='text'
				minlength={1}
				maxlength={63}
				onSlInput={onSettingInput}
				value={settings?.username || ''}
			/>
			<SlInput
				name='password'
				label='Password'
				placeholder='password'
				type='password'
				minlength={1}
				maxlength={100}
				onSlInput={onSettingInput}
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
				onSlInput={onSettingInput}
				value={settings?.database || ''}
			/>
			<SlInput
				name='schema'
				label='Schema'
				placeholder='public'
				type='text'
				minlength={1}
				maxlength={63}
				onSlInput={onSettingInput}
				value={settings?.schema || ''}
			/>
		</>
	);
};

export { PostgreSQLSettings };
