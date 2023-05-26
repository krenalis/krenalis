import * as variants from './variants';
import * as icons from './icons';

const statuses = {
	connectionDoesNotExistAnymore: [variants.DANGER, icons.NOT_FOUND, 'The connection does not exist anymore'],
	connectorDoesNotExistAnymore: [variants.DANGER, icons.NOT_FOUND, 'The connector does not exist anymore'],
	workspaceDoesNotExistAnymore: [variants.DANGER, icons.NOT_FOUND, 'The workspace does not exist anymore'],
	tooManyKeys: [variants.DANGER, icons.FORBIDDEN, 'You have reached the maximum number of keys'],
	uniqueKey: [variants.DANGER, icons.FORBIDDEN, 'You cannot delete the last key'],
	alreadyInProgress: [variants.DANGER, icons.FORBIDDEN, 'You already have an import in progress'],
	noStorage: [variants.DANGER, icons.FORBIDDEN, 'You must associate a storage connection before starting an import'],
	noTransformationNorMappings: [
		variants.DANGER,
		icons.FORBIDDEN,
		'You must associate the mappings or a transformation before starting an import',
	],
	noWarehouse: [
		variants.DANGER,
		icons.FORBIDDEN,
		'You must connect a data warehouse to the workspace before starting an import',
	],
	warehouseNotConnected: [variants.DANGER, icons.NOT_FOUND, 'The workspace is not connected to any data warehouse'],
	warehouseConnectionFailed: [variants.DANGER, icons.NOT_FOUND, 'The connection to the data warehouse has failed'],
	notEnabled: [variants.DANGER, icons.FORBIDDEN, 'You must enable the connection before starting an import'],
	storageNotEnabled: [
		variants.DANGER,
		icons.FORBIDDEN,
		'You must enable the storage associated to this connection before starting an import',
	],
	invalidSchemaTable: [variants.DANGER, icons.INVALID_INSERTED_VALUE, 'The schema table is invalid'],
	storageNotExist: [variants.DANGER, icons.NOT_FOUND, 'The selected storage does not exist anymore'],
	propertyNotExist: [variants.DANGER, icons.NOT_FOUND, 'One of the schema properties does not exist'],
	settingsNotValid: [variants.DANGER, icons.INVALID_INSERTED_VALUE, 'These settings are not valid'],
	listenerDoesNotExist: [variants.DANGER, icons.NOT_FOUND, 'The listener does not exist'],
	tooManyListeners: [variants.DANGER, icons.FORBIDDEN, 'You have exceeded the number of event listeners allowed'],
	alreadyHasTransformation: [
		variants.DANGER,
		icons.FORBIDDEN,
		'This connection already has a configured transformation',
	],
	alreadyHasMappings: [variants.DANGER, icons.FORBIDDEN, 'This connection already has configured mappings'],
	noUsersSchema: [variants.DANGER, icons.NOT_FOUND, 'The user schema is not currently defined'],
	noGroupsSchema: [variants.DANGER, icons.NOT_FOUND, 'The groups schema is not currently defined'],
	eventTypeNotExists: [variants.DANGER, icons.NOT_FOUND, 'This event type does not exist enymore'],
	actionExecutionInProgress: [variants.DANGER, icons.FORBIDDEN, 'This action is already in progress'],
	querySet: [variants.PRIMARY, icons.OK, 'Your query has been successfully saved'],
	mappingsSaved: [variants.SUCCESS, icons.OK, 'Your mappings have been successfully saved'],
	schemasReloaded: [variants.SUCCESS, icons.OK, 'The schemas have been reloaded successfully'],
	schemaLoaded: [variants.SUCCESS, icons.OK, 'The schema has been loaded successfully'],
	transformationSaved: [variants.SUCCESS, icons.OK, 'Your transformation has been successfully saved'],
	transformationCleanedUp: [variants.SUCCESS, icons.OK, 'Your transformation has been successfully cleaned up'],
	actionSaved: [variants.SUCCESS, icons.OK, 'Your action has been successfully saved'],
	connectionReloaded: [variants.SUCCESS, icons.OK, 'The connection has been reloaded successfully'],
	connectionSaved: [variants.SUCCESS, icons.OK, 'The connection settings have been saved successfully'],
};

export default statuses;
