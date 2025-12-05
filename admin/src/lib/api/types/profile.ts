interface ProfileEventContextApp {
	name?: string;
	version?: string;
	build?: string;
	namespace?: string;
}

interface ProfileEventContextBrowser {
	name?: string;
	other?: string;
	version?: string;
}

interface ProfileEventContextCampaign {
	name?: string;
	source?: string;
	medium?: string;
	term?: string;
	content?: string;
}

interface ProfileEventContextDevice {
	id?: string;
	advertisingId?: string;
	adTrackingEnabled?: boolean;
	manufacturer?: string;
	model?: string;
	name?: string;
	type?: string;
	token?: string;
}

interface ProfileEventContextLibrary {
	name?: string;
	version?: string;
}

interface ProfileEventContextLocation {
	city?: string;
	country?: string;
	latitude?: number;
	longitude?: number;
	speed?: number;
}

interface ProfileEventContextNetwork {
	bluetooth?: boolean;
	carrier?: string;
	cellular?: boolean;
	wifi?: boolean;
}

interface ProfileEventContextOS {
	name?: string;
	other?: string;
	version?: string;
}

interface ProfileEventContextPage {
	path?: string;
	referrer?: string;
	search?: string;
	title?: string;
	url?: string;
}

interface ProfileEventContextReferrer {
	id?: string;
	type?: string;
}

interface ProfileEventContextScreen {
	width?: number;
	height?: number;
	density?: number;
}

interface ProfileEventContextSession {
	sessionId?: number;
	sessionStart?: boolean;
}

interface ProfileEventContext {
	app?: ProfileEventContextApp;
	browser?: ProfileEventContextBrowser;
	campaign?: ProfileEventContextCampaign;
	device?: ProfileEventContextDevice;
	ip?: string;
	library?: ProfileEventContextLibrary;
	locale?: string;
	location?: ProfileEventContextLocation;
	network?: ProfileEventContextNetwork;
	os?: ProfileEventContextOS;
	page?: ProfileEventContextPage;
	referrer?: ProfileEventContextReferrer;
	screen?: ProfileEventContextScreen;
	session?: ProfileEventContextSession;
	timezone?: string;
	userAgent?: string;
}

interface ProfileEvent {
	id?: string;
	mpid?: string;
	connectionId?: number;
	anonymousId?: string;
	category?: string;
	context?: ProfileEventContext;
	event?: string;
	groupId?: string;
	messageId?: string;
	name?: string;
	properties?: any;
	receivedAt?: string;
	sentAt?: string;
	originalTimestamp?: number;
	timestamp?: string;
	traits?: any;
	type?: string;
	previousId?: string;
	userId?: string;
}

type ProfileAttributes = Record<string, any>;

interface Identity {
	userId: string;
	anonymousIds: string[] | null;
	lastChangeTime: string;
	pipeline: number;
	connection: number;
}

interface Profile {
	id: number;
	events: ProfileEvent[];
	attributes: ProfileAttributes;
}

export type { Profile, ProfileEvent, ProfileAttributes, Identity };
