import { SpecialProperties } from '../../types/internal/app';

const firstNameIdentifiers = [
	'firstname',
	'Firstname',
	'FirstName',
	'first_name',
	'First_Name',
	'First_name',
	'FIRSTNAME',
	'FIRST_NAME',
];

const lastNameIdentifiers = [
	'lastname',
	'Lastname',
	'LastName',
	'last_name',
	'Last_Name',
	'Last_name',
	'LASTNAME',
	'LAST_NAME',
];

const emailIdentifiers = ['email', 'Email', 'EMail', 'e_mail', 'E_mail', 'EMAIL'];

const idIdentifiers = ['id', 'ID'];

const extractSpecialProperties = (resources: Record<string, any>[]): SpecialProperties => {
	let firstNameID: string, lastNameID: string, emailID: string, idID: string;
	const keys = Object.keys(resources[0]);
	for (const key of keys) {
		if (firstNameIdentifiers.includes(key) || firstNameIdentifiers.includes(resources[0][key].property.label)) {
			firstNameID = key;
			continue;
		}
		if (lastNameIdentifiers.includes(key) || lastNameIdentifiers.includes(resources[0][key].property.label)) {
			lastNameID = key;
			continue;
		}
		if (emailIdentifiers.includes(key) || emailIdentifiers.includes(resources[0][key].property.label)) {
			emailID = key;
			continue;
		}
		if (idIdentifiers.includes(key) || idIdentifiers.includes(resources[0][key].property.label)) {
			idID = key;
			continue;
		}
	}
	return {
		firstNameID,
		lastNameID,
		emailID,
		idID,
	};
};

export default extractSpecialProperties;
