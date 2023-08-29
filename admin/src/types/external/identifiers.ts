import { Mapping } from './action';

type ActionIdentifiers = string[];

interface AnonymousIdentifiers {
	Priority: string[];
	Mapping: Mapping;
}

export { ActionIdentifiers, AnonymousIdentifiers };
