import { useContext, useMemo } from 'react';
import AppContext from '../context/AppContext';
import { flattenSchema } from '../lib/helpers/transformedAction';
import { TransformedIdentifiers, isTypeSupportedAsIdentifier } from '../lib/helpers/transformedIdentifiers';
import { ObjectType, Property } from '../types/external/types';

const MAPPED_INDEX = 0;
const IDENTIFIER_INDEX = 1;

const useIdentifiersMapping = (
	mapping: TransformedIdentifiers,
	setMapping:
		| React.Dispatch<React.SetStateAction<TransformedIdentifiers | undefined>>
		| ((identifiers: TransformedIdentifiers) => void),
	inputSchema: ObjectType,
	outputSchema: ObjectType,
	onRemoveIdentifier?: (identifier: string) => void,
) => {
	const { api, showError } = useContext(AppContext);

	const flatOutputSchema = useMemo(() => flattenSchema(outputSchema), [outputSchema]);

	// unusableProperties contains the list of properties that cannot be used as
	// identifiers.
	const unusableProperties: string[] = useMemo(() => {
		const nonSupported: string[] = [];
		for (const propertyName in flatOutputSchema) {
			const typeName = flatOutputSchema[propertyName].full.type.name;
			const isSupported = isTypeSupportedAsIdentifier(typeName);
			if (!isSupported) {
				nonSupported.push(propertyName);
			}
		}
		return nonSupported;
	}, [flatOutputSchema]);

	// usedProperties contains the list of properties already used as
	// identifiers.
	const usedProperties = useMemo(
		() =>
			mapping.map(([, outputProperty]) => {
				if (outputProperty.value !== '') {
					return outputProperty.value;
				} else {
					return null;
				}
			}),
		[mapping],
	);

	const nonSelectableProperties = useMemo(
		() => [...unusableProperties, ...usedProperties] as string[],
		[unusableProperties, usedProperties],
	);

	const validateExpression = async (expression: string, properties: Property[], destinationProperty: Property) => {
		let message = '';
		if (expression !== '') {
			try {
				message = await api.validateExpression(
					expression,
					properties,
					destinationProperty.type,
					destinationProperty.required,
					destinationProperty.nullable,
				);
			} catch (err) {
				showError(err);
				return;
			}
		}
		return message;
	};

	const updateMappedProperty = async (pos: number, value: string) => {
		const m = [...mapping];
		const i = pos - 1;
		m[i][MAPPED_INDEX].error = ''; // reset the error.
		const associatedIdentifier = m[i][IDENTIFIER_INDEX];
		const destinationProperty = flatOutputSchema![associatedIdentifier.value];
		if (destinationProperty) {
			const errorMessage = await validateExpression(value, inputSchema.properties, destinationProperty.full);
			m[i][MAPPED_INDEX].error = errorMessage as string;
		}
		m[i][MAPPED_INDEX].value = value;
		setMapping(m);
	};

	const updateIdentifier = async (pos: number, value: string) => {
		const m = [...mapping];
		const i = pos - 1;
		m[i][IDENTIFIER_INDEX].value = value;
		setMapping(m);
		const mapped = m[i][MAPPED_INDEX].value;
		await updateMappedProperty(pos, mapped);
	};

	const moveAssociationUp = (position: number) => {
		const elementIndex = position - 1;
		const element = mapping[elementIndex];
		const previousElementIndex = elementIndex - 1;
		const previousElement = mapping[previousElementIndex];
		const m = [
			...mapping.slice(0, previousElementIndex),
			element,
			previousElement,
			...mapping.slice(elementIndex + 1),
		];
		setMapping(m);
	};

	const moveAssociationDown = (position: number) => {
		const elementIndex = position - 1;
		const element = mapping[elementIndex];
		const nextElementIndex = elementIndex + 1;
		const nextElement = mapping[nextElementIndex];
		const m = [...mapping.slice(0, elementIndex), nextElement, element, ...mapping.slice(nextElementIndex + 1)];
		setMapping(m);
	};

	const removeAssociation = (position: number) => {
		const m = [...mapping];
		if (onRemoveIdentifier) {
			onRemoveIdentifier(m[position - 1][1].value);
		}
		m.splice(position - 1, 1);
		setMapping(m);
	};

	const addAssociation = () => {
		const m = [...mapping];
		m.push([{ value: '', error: '' }, { value: '' }]);
		setMapping(m);
	};

	return {
		nonSelectableProperties,
		updateMappedProperty,
		updateIdentifier,
		moveAssociationUp,
		moveAssociationDown,
		removeAssociation,
		addAssociation,
	};
};

export default useIdentifiersMapping;
