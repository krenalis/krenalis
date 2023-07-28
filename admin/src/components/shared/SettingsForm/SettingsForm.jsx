import './SettingsForm.css';
import { SettingsContext } from '../../../context/SettingsContext';

const SettingsForm = ({ fields, actions, values, onChange }) => {
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
