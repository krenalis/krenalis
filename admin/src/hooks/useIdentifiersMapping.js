import { useContext, useMemo } from 'react';
import { AppContext } from '../context/providers/AppProvider';
import { flattenSchema } from '../lib/helpers/action';
import { isTypeSupportedAsIdentifier } from '../lib/helpers/identifiers';

const MAPPED_INDEX = 0;
const IDENTIFIER_INDEX = 1;

const useIdentifiersMapping = (mapping, setMapping, inputSchema, outputSchema, onRemoveIdentifier) => {
	const { api, showError } = useContext(AppContext);

	const flatOutputSchema = useMemo(() => flattenSchema(outputSchema), [outputSchema]);

	// unusableProperties contains the list of properties that cannot be used as
	// identifiers.
	const unusableProperties = useMemo(() => {
		const nonSupported = [];
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
		() => mapping.map(([, outputProperty]) => outputProperty.value !== '' && outputProperty.value),
		[mapping]
	);

	const nonSelectableProperties = useMemo(
		() => [...unusableProperties, ...usedProperties],
		[unusableProperties, usedProperties]
	);

	const isRemoveButtonDisabled = useMemo(() => {
		if (mapping.length === 1) {
			const isFirstMappedPropertyEmpty = mapping[0][0].value === '';
			const isFirstIdentifierEmpty = mapping[0][1].value === '';
			if (isFirstMappedPropertyEmpty && isFirstIdentifierEmpty) return true;
		}
		return false;
	}, [mapping]);

	const validateExpression = async (expression, schema, destinationProperty) => {
		let message = '';
		if (expression !== '') {
			try {
				message = await api.validateExpression(
					expression,
					schema,
					destinationProperty.type,
					destinationProperty.nullable
				);
			} catch (err) {
				showError(err);
				return;
			}
		}
		return message;
	};

	const updateMappedProperty = async (pos, value) => {
		const m = [...mapping];
		const i = pos - 1;
		m[i][MAPPED_INDEX].error = ''; // reset the error.
		const associatedIdentifier = m[i][IDENTIFIER_INDEX];
		const destinationProperty = flatOutputSchema[associatedIdentifier.value];
		if (destinationProperty && !associatedIdentifier.error) {
			const errorMessage = await validateExpression(value, inputSchema, destinationProperty.full);
			m[i][MAPPED_INDEX].error = errorMessage;
		}
		m[i][MAPPED_INDEX].value = value;
		setMapping(m);
	};

	const updateIdentifier = async (pos, value) => {
		const m = [...mapping];
		const i = pos - 1;
		m[i][IDENTIFIER_INDEX].error = ''; // reset the error.
		const isTypeSupported = !unusableProperties.includes(value);
		if (!isTypeSupported) {
			m[i][
				IDENTIFIER_INDEX
			].error = `Type ${flatOutputSchema[value].full.type.name} is not supported as identifier`;
		}
		m[i][IDENTIFIER_INDEX].value = value;
		setMapping(m);
		const mapped = m[i][MAPPED_INDEX].value;
		await updateMappedProperty(pos, mapped);
	};

	const moveAssociationUp = (position) => {
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

	const moveAssociationDown = (position) => {
		const elementIndex = position - 1;
		const element = mapping[elementIndex];
		const nextElementIndex = elementIndex + 1;
		const nextElement = mapping[nextElementIndex];
		const m = [...mapping.slice(0, elementIndex), nextElement, element, ...mapping.slice(nextElementIndex + 1)];
		setMapping(m);
	};

	const removeAssociation = (position) => {
		const m = [...mapping];
		if (onRemoveIdentifier) {
			onRemoveIdentifier(m[position - 1][1].value);
		}
		if (m.length === 1) {
			m[0][0].value = '';
			m[0][1].value = '';
		} else {
			m.splice(position - 1, 1);
		}
		setMapping(m);
	};

	const addAssociation = () => {
		const m = [...mapping];
		m.push([{ value: '', error: '' }, { value: '' }]);
		setMapping(m);
	};

	return {
		nonSelectableProperties,
		isRemoveButtonDisabled,
		updateMappedProperty,
		updateIdentifier,
		moveAssociationUp,
		moveAssociationDown,
		removeAssociation,
		addAssociation,
	};
};

export default useIdentifiersMapping;
