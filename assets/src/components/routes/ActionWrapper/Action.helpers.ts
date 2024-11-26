import { TransformedAction, TransformedMapping } from '../../../lib/core/action';
import { SampleIdentifiers } from './Action.types';

const updateMappingProperty = (action: TransformedAction, name: string, value: string, error: string) => {
	const getAlternativeProperties = (name: string, mapping: TransformedMapping): string[] => {
		const indentation = mapping[name].indentation;
		const parentProperties: string[] = [];
		for (const k in mapping) {
			if (mapping[k].indentation! < indentation! && name.startsWith(k)) {
				parentProperties.push(k);
			}
		}
		const childrenProperties: string[] = [];
		for (const k in mapping) {
			if (mapping[k].indentation! > indentation! && k.startsWith(name)) {
				childrenProperties.push(k);
			}
		}
		return [...parentProperties, ...childrenProperties];
	};

	const a = { ...action };

	if (a.transformation.mapping == null) return a;

	if (a.transformation.mapping[name].value === '' && value !== '') {
		const alternativeProperties = getAlternativeProperties(name, a.transformation.mapping);
		// disable
		for (const k in a.transformation.mapping) {
			if (alternativeProperties.includes(k)) {
				a.transformation.mapping[k].disabled = true;
			}
		}
	} else if (value === '') {
		let hasFilledSiblings = false;
		const { root, indentation } = a.transformation.mapping[name];
		for (const k in a.transformation.mapping) {
			if (
				k !== name &&
				a.transformation.mapping[k].root === root &&
				a.transformation.mapping[k].indentation === indentation &&
				a.transformation.mapping[k].value !== ''
			) {
				hasFilledSiblings = true;
			}
		}
		if (!hasFilledSiblings) {
			// enable
			const alternativeProperties = getAlternativeProperties(name, a.transformation.mapping);
			for (const k in a.transformation.mapping) {
				if (alternativeProperties.includes(k)) {
					a.transformation.mapping[k].disabled = false;
				}
			}
		}
	}

	a.transformation.mapping[name].error = error;
	a.transformation.mapping[name].value = value;
	return a;
};

const checkIfPropertyExists = (property: string, schema: TransformedMapping): string => {
	if (schema == null || property === '' || property == null) {
		return '';
	}
	if (schema[property] == null) {
		return `Property "${property}" does not exist`;
	}
	return '';
};

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

const idIdentifiers = ['id', 'ID', 'Id', '__id__'];

// getSampleIdentifiers returns the names of the properties that are used in the
// UI to identify a sample.
const getSampleIdentifiers = (sample: Record<string, any>): SampleIdentifiers | null => {
	let firstNameIdentifier: string, lastNameIdentifier: string, emailIdentifier: string, idIdentifier: string;
	for (const key in sample) {
		if (firstNameIdentifiers.includes(key)) {
			firstNameIdentifier = key;
			continue;
		}
		if (lastNameIdentifiers.includes(key)) {
			lastNameIdentifier = key;
			continue;
		}
		if (emailIdentifiers.includes(key)) {
			emailIdentifier = key;
			continue;
		}
		if (idIdentifiers.includes(key)) {
			idIdentifier = key;
			continue;
		}
	}
	return {
		firstNameIdentifier,
		lastNameIdentifier,
		emailIdentifier,
		idIdentifier,
	};
};

export { updateMappingProperty, checkIfPropertyExists, getSampleIdentifiers };
