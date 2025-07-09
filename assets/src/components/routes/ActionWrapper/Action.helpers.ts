import { getHierarchicalPaths, TransformedAction, TransformedMapping } from '../../../lib/core/action';
import { SampleIdentifiers } from './Action.types';

const updateMappingProperty = (
	action: TransformedAction,
	path: string,
	value: string,
	error: string,
): TransformedAction => {
	const a = { ...action };
	const mapping = a.transformation.mapping;
	if (mapping == null) {
		return a;
	}

	const oldValue = mapping[path].value;

	// Update the mapping.
	mapping[path].error = error;
	mapping[path].value = value;

	// Enable/disable the properties in the hierarchy.
	const { ancestors, descendants } = getHierarchicalPaths(path, mapping);
	const wasFilled = oldValue === '' && value !== '';
	const wasCleared = value === '';
	if (wasFilled) {
		// Disable the properties in the hierarchy.
		for (const p of [...ancestors, ...descendants]) {
			mapping[p].disabled = true;
		}
	} else if (wasCleared) {
		// Enable the descendants.
		for (const p of descendants) {
			mapping[p].disabled = false;
		}

		// Enable the ancestors, but only those that do not have other
		// filled descendants.
		for (const a of ancestors) {
			const { descendants: desc } = getHierarchicalPaths(a, mapping);
			let hasFilledDescendants = false;
			for (const d of desc) {
				if (mapping[d].value !== '') {
					hasFilledDescendants = true;
					break;
				}
			}
			if (!hasFilledDescendants) {
				mapping[a].disabled = false;
			}
		}
	}

	a.transformation.mapping = mapping;

	return a;
};

const updateMappingPropertyError = (action: TransformedAction, path: string, error: string): TransformedAction => {
	const a = { ...action };
	const mapping = a.transformation.mapping;
	if (mapping == null) {
		return a;
	}
	mapping[path].error = error;
	a.transformation.mapping = mapping;
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

export { updateMappingProperty, updateMappingPropertyError, checkIfPropertyExists, getSampleIdentifiers };
