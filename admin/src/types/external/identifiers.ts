import { Mapping } from './action';

type Identifiers = string[];

interface AnonymousIdentifiers {
	Priority: string[];
	Mapping: Mapping;
}

export type { Identifiers, AnonymousIdentifiers };
