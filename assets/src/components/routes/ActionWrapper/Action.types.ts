import { Property } from '../../../lib/api/types/types';

interface SpecialProperties {
	firstNameID: string;
	lastNameID: string;
	emailID: string;
	idID: string;
}

interface SampleProperty {
	value: any;
	property: Property;
}

type Sample = Record<string, SampleProperty>;

export { Sample, SpecialProperties };
