interface SampleIdentifiers {
	firstNameIdentifier: string;
	lastNameIdentifier: string;
	emailIdentifier: string;
	idIdentifier: string;
}

type Sample = Record<string, any>;

export { Sample, SampleIdentifiers };
