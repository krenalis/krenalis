const WORKSPACE_ID_KEY = 'meergo_admin_workspace_id';
const ADD_CONNECTOR_CODE_KEY = 'meergo_admin_add_connector_code';
const ADD_CONNECTION_ROLE_KEY = 'meergo_admin_add_connection_role';
const ADD_CONNECTION_ID_KEY = 'meergo_admin_add_connection_id';
const IS_PASSWORDLESS_KEY = 'meergo_admin_is_passwordless';
const IS_DOCKER_KEY = 'meergo_admin_is_docker';
const PROFILES_TAB_KEY = 'meergo_admin_profiles_tab';
const PROFILES_EXPANDED_ATTRIBUTES_KEY = 'meergo_admin_PROFILES_expanded_attributes';
const PROFILES_PROPERTIES_KEY = 'meergo_admin_profiles_properties';

// storageKeysToBeRemoved contains the list of keys in browser localStorage that
// can be cleared when resetting the client state, for example to attempt fixing
// an unhandled error in the UI.
const storageKeysToBeRemoved = [
	ADD_CONNECTOR_CODE_KEY,
	ADD_CONNECTION_ROLE_KEY,
	ADD_CONNECTION_ID_KEY,
	IS_PASSWORDLESS_KEY,
	IS_DOCKER_KEY,
	PROFILES_TAB_KEY,
	PROFILES_EXPANDED_ATTRIBUTES_KEY,
	PROFILES_PROPERTIES_KEY,
];

export {
	WORKSPACE_ID_KEY,
	ADD_CONNECTOR_CODE_KEY,
	ADD_CONNECTION_ROLE_KEY,
	ADD_CONNECTION_ID_KEY,
	IS_PASSWORDLESS_KEY,
	IS_DOCKER_KEY,
	PROFILES_TAB_KEY,
	PROFILES_EXPANDED_ATTRIBUTES_KEY,
	PROFILES_PROPERTIES_KEY,
	storageKeysToBeRemoved,
};
