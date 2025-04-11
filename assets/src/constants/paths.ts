const UI_BASE_PATH = '/ui/';
const SIGN_UP_PATH = `${UI_BASE_PATH}sign-up`;
const RESET_PASSWORD_PATH = `${UI_BASE_PATH}reset-password`;
const FULLSCREEN_PATHS = [
	UI_BASE_PATH,
	`${UI_BASE_PATH}sign-up/:token`,
	RESET_PASSWORD_PATH,
	`${UI_BASE_PATH}reset-password/:token`,
	`${UI_BASE_PATH}workspaces`,
	`${UI_BASE_PATH}workspaces/add`,
	`${UI_BASE_PATH}connections/:id/actions/edit/:action`,
	`${UI_BASE_PATH}connections/:id/actions/add/event/:eventType`,
	`${UI_BASE_PATH}connections/:id/actions/add/event`,
	`${UI_BASE_PATH}connections/:id/actions/add/:actionTarget`,
	`${UI_BASE_PATH}schema/edit`,
];

export { UI_BASE_PATH, SIGN_UP_PATH, RESET_PASSWORD_PATH, FULLSCREEN_PATHS };
