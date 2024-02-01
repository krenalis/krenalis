import { Identifiers } from '../../types/external/identifiers';
import { TransformedMapping } from './transformedAction';

const SUPPORTED_IDENTIFIERS_TYPES = ['Int', 'Uint', 'Decimal', 'UUID', 'Inet', 'Text', 'Array'];

const DEFAULT_IDENTIFIERS_MAPPING = [];

interface Mapped {
	value: string;
	error: string;
}

interface Identifier {
	value: string;
}

type IdentifierAssociation = [Mapped, Identifier];

type TransformedIdentifiers = IdentifierAssociation[];

const isTypeSupportedAsIdentifier = (type: string): boolean => {
	if (SUPPORTED_IDENTIFIERS_TYPES.includes(type)) {
		return true;
	}
	return false;
};

const transformIdentifiers = (identifiers: Identifiers, mapping: TransformedMapping): TransformedIdentifiers => {
	return identifiers.map((identifier) => [{ value: mapping[identifier].value, error: '' }, { value: identifier }]);
};

const untransformIdentifiers = (transformed: TransformedIdentifiers): Identifiers => {
	return transformed.map(([, identifier]) => identifier.value);
};

export { DEFAULT_IDENTIFIERS_MAPPING, isTypeSupportedAsIdentifier, transformIdentifiers, untransformIdentifiers };

export type { TransformedIdentifiers, IdentifierAssociation };
