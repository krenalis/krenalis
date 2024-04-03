import * as variants from './variants';
import * as icons from './icons';
import { Status } from '../types/internal/app';

const statuses: Record<string, Status> = {
	connectionDoesNotExistAnymore: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'The connection does not exist anymore',
	},
	connectorDoesNotExistAnymore: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'The connector does not exist anymore',
	},
	workspaceDoesNotExistAnymore: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'The workspace does not exist anymore',
	},
	usersNotFound: { variant: variants.DANGER, icon: icons.NOT_FOUND, text: 'This users does not exist' },
	tooManyKeys: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'The maximum number of keys has been reached',
	},
	uniqueKey: { variant: variants.DANGER, icon: icons.FORBIDDEN, text: 'The last key cannot be deleted' },
	alreadyInProgress: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'There is already an import in progress',
	},
	noTransformationNorMappings: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'Please associate the mappings or a transformation before starting an import',
	},
	noWarehouse: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'The workspace is not connected to any data warehouse',
	},
	warehouseNotConnected: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'The workspace is not connected to any data warehouse',
	},
	dataWarehouseFailed: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'An error occurred with the data warehouse',
	},
	notEnabled: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'Please ensure that the connection is enabled before starting an import',
	},
	storageNotEnabled: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'Please enable the storage associated with this connection before starting an import',
	},
	storageNotExist: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'The selected storage does not exist anymore',
	},
	propertyNotExist: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'One of the schema properties does not exist',
	},
	settingsNotValid: {
		variant: variants.DANGER,
		icon: icons.INVALID_INSERTED_VALUE,
		text: 'These settings are not valid',
	},
	listenerDoesNotExist: { variant: variants.DANGER, icon: icons.NOT_FOUND, text: 'The listener does not exist' },
	tooManyListeners: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'Please note that the number of event listeners allowed has been exceeded',
	},
	alreadyHasTransformation: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'This connection already has a configured transformation',
	},
	alreadyHasMappings: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'This connection already has configured mappings',
	},
	eventTypeNotExist: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'This event type does not exist anymore',
	},
	actionExecutionInProgress: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'This action is already in progress',
	},
	querySet: { variant: variants.PRIMARY, icon: icons.OK, text: 'The query has been saved' },
	mappingsSaved: { variant: variants.SUCCESS, icon: icons.OK, text: 'The mappings have been saved' },
	schemasReloaded: { variant: variants.SUCCESS, icon: icons.OK, text: 'The schemas have been reloaded' },
	schemaLoaded: { variant: variants.SUCCESS, icon: icons.OK, text: 'The schema has been loaded' },
	transformationSaved: {
		variant: variants.SUCCESS,
		icon: icons.OK,
		text: 'The transformation has been saved',
	},
	transformationCleanedUp: {
		variant: variants.SUCCESS,
		icon: icons.OK,
		text: 'The transformation has been cleaned up',
	},
	actionSaved: { variant: variants.SUCCESS, icon: icons.OK, text: 'The action has been saved' },
	connectionReloaded: {
		variant: variants.SUCCESS,
		icon: icons.OK,
		text: 'The connection has been reloaded',
	},
	connectionSaved: {
		variant: variants.SUCCESS,
		icon: icons.OK,
		text: 'The connection settings have been saved',
	},
	linkedStorageDoesNotExistAnymore: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'The storage of this file connection does not exist anymore',
	},
};

export default statuses;
