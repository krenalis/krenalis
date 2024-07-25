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

	if (a.Transformation.Mapping == null) return a;

	if (a.Transformation.Mapping[name].value === '' && value !== '') {
		const alternativeProperties = getAlternativeProperties(name, a.Transformation.Mapping);
		// disable
		for (const k in a.Transformation.Mapping) {
			if (alternativeProperties.includes(k)) {
				a.Transformation.Mapping[k].disabled = true;
			}
		}
	} else if (value === '') {
		let hasFilledSiblings = false;
		const { root, indentation } = a.Transformation.Mapping[name];
		for (const k in a.Transformation.Mapping) {
			if (
				k !== name &&
				a.Transformation.Mapping[k].root === root &&
				a.Transformation.Mapping[k].indentation === indentation &&
				a.Transformation.Mapping[k].value !== ''
			) {
				hasFilledSiblings = true;
			}
		}
		if (!hasFilledSiblings) {
			// enable
			const alternativeProperties = getAlternativeProperties(name, a.Transformation.Mapping);
			for (const k in a.Transformation.Mapping) {
				if (alternativeProperties.includes(k)) {
					a.Transformation.Mapping[k].disabled = false;
				}
			}
		}
	}

	a.Transformation.Mapping[name].error = error;
	a.Transformation.Mapping[name].value = value;
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
		if (!sample.hasOwnProperty(key)) {
			continue;
		}
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
