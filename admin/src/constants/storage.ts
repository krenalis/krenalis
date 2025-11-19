const WORKSPACE_ID_KEY = 'meergo_admin_workspace_id';
const ADD_CONNECTOR_CODE_KEY = 'meergo_admin_add_connector_code';
const ADD_CONNECTION_ROLE_KEY = 'meergo_admin_add_connection_role';
const ADD_CONNECTION_ID_KEY = 'meergo_admin_add_connection_id';
const IS_PASSWORDLESS_KEY = 'meergo_admin_is_passwordless';
const IS_DOCKER_KEY = 'meergo_admin_is_docker';
const USERS_TAB_KEY = 'meergo_admin_users_tab';
const USERS_EXPANDED_TRAITS_KEY = 'meergo_admin_users_expanded_traits';
const USERS_PROPERTIES_KEY = 'meergo_admin_users_properties';

// storageKeysToBeRemoved contains the list of keys in browser localStorage that
// can be cleared when resetting the client state, for example to attempt fixing
// an unhandled error in the UI.
const storageKeysToBeRemoved = [
	ADD_CONNECTOR_CODE_KEY,
	ADD_CONNECTION_ROLE_KEY,
	ADD_CONNECTION_ID_KEY,
	IS_PASSWORDLESS_KEY,
	IS_DOCKER_KEY,
	USERS_TAB_KEY,
	USERS_EXPANDED_TRAITS_KEY,
	USERS_PROPERTIES_KEY,
];

export {
	WORKSPACE_ID_KEY,
	ADD_CONNECTOR_CODE_KEY,
	ADD_CONNECTION_ROLE_KEY,
	ADD_CONNECTION_ID_KEY,
	IS_PASSWORDLESS_KEY,
	IS_DOCKER_KEY,
	USERS_TAB_KEY,
	USERS_EXPANDED_TRAITS_KEY,
	USERS_PROPERTIES_KEY,
	storageKeysToBeRemoved,
};
