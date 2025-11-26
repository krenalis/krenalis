const UI_BASE_PATH = '/admin/';
const SIGN_UP_PATH = `${UI_BASE_PATH}sign-up`;
const RESET_PASSWORD_PATH = `${UI_BASE_PATH}reset-password`;
const FULLSCREEN_PATHS = [
	UI_BASE_PATH,
	`${UI_BASE_PATH}sign-up/:token`,
	RESET_PASSWORD_PATH,
	`${UI_BASE_PATH}reset-password/:token`,
	`${UI_BASE_PATH}workspaces`,
	`${UI_BASE_PATH}workspaces/create`,
	`${UI_BASE_PATH}connections/:id/pipelines/edit/:pipeline`,
	`${UI_BASE_PATH}connections/:id/pipelines/add/event/:eventType`,
	`${UI_BASE_PATH}connections/:id/pipelines/add/event`,
	`${UI_BASE_PATH}connections/:id/pipelines/add/:pipelineTarget`,
	`${UI_BASE_PATH}schema/edit`,
];
const CONNECTORS_ASSETS_PATH = 'connectors';
const WAREHOUSES_ASSETS_PATH = 'warehouses';

export {
	UI_BASE_PATH,
	SIGN_UP_PATH,
	RESET_PASSWORD_PATH,
	FULLSCREEN_PATHS,
	CONNECTORS_ASSETS_PATH,
	WAREHOUSES_ASSETS_PATH,
};
