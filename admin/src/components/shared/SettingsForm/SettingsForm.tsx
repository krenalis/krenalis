import React, { ReactNode } from 'react';
import './SettingsForm.css';
import { SettingsContext } from '../../../context/SettingsContext';
import { UIValues } from '../../../types/external/api';

interface SettingsFormProps {
	fields: ReactNode[];
	actions: ReactNode[];
	values: UIValues;
	onChange: (name: string, value: any) => void;
}

const SettingsForm = ({ fields, actions, values, onChange }: SettingsFormProps) => {
	return (
		<div className='settings-form'>
			<SettingsContext.Provider value={{ values, onChange }}>
				<div className='settings-form__fields'>{fields}</div>
			</SettingsContext.Provider>
			<div className='settings-form__actions'>{actions}</div>
		</div>
	);
};

export default SettingsForm;
