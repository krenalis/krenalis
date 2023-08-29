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
		text: 'You have reached the maximum number of keys',
	},
	uniqueKey: { variant: variants.DANGER, icon: icons.FORBIDDEN, text: 'You cannot delete the last key' },
	alreadyInProgress: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'You already have an import in progress',
	},
	noStorage: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'You must associate a storage connection before starting an import',
	},
	noTransformationNorMappings: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'You must associate the mappings or a transformation before starting an import',
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
	warehouseConnectionFailed: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'The connection to the data warehouse has failed',
	},
	notEnabled: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'You must enable the connection before starting an import',
	},
	storageNotEnabled: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'You must enable the storage associated to this connection before starting an import',
	},
	invalidSchemaTable: {
		variant: variants.DANGER,
		icon: icons.INVALID_INSERTED_VALUE,
		text: 'The schema table is invalid',
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
		text: 'You have exceeded the number of event listeners allowed',
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
	eventTypeNotExists: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'This event type does not exist enymore',
	},
	actionExecutionInProgress: {
		variant: variants.DANGER,
		icon: icons.FORBIDDEN,
		text: 'This action is already in progress',
	},
	querySet: { variant: variants.PRIMARY, icon: icons.OK, text: 'Your query has been successfully saved' },
	mappingsSaved: { variant: variants.SUCCESS, icon: icons.OK, text: 'Your mappings have been successfully saved' },
	schemasReloaded: { variant: variants.SUCCESS, icon: icons.OK, text: 'The schemas have been reloaded successfully' },
	schemaLoaded: { variant: variants.SUCCESS, icon: icons.OK, text: 'The schema has been loaded successfully' },
	transformationSaved: {
		variant: variants.SUCCESS,
		icon: icons.OK,
		text: 'Your transformation has been successfully saved',
	},
	transformationCleanedUp: {
		variant: variants.SUCCESS,
		icon: icons.OK,
		text: 'Your transformation has been successfully cleaned up',
	},
	actionSaved: { variant: variants.SUCCESS, icon: icons.OK, text: 'Your action has been successfully saved' },
	connectionReloaded: {
		variant: variants.SUCCESS,
		icon: icons.OK,
		text: 'The connection has been reloaded successfully',
	},
	connectionSaved: {
		variant: variants.SUCCESS,
		icon: icons.OK,
		text: 'The connection settings have been saved successfully',
	},
	linkedStorageDoesNotExistAnymore: {
		variant: variants.DANGER,
		icon: icons.NOT_FOUND,
		text: 'The storage of this file connection does not exist anymore',
	},
};

export default statuses;
