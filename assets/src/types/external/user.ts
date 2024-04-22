interface UserEventContextApp {
	name: string;
	version: string;
	build: string;
	namespace: string;
}

interface UserEventContextBrowser {
	name: string;
	other: string;
	version: string;
}

interface UserEventContextCampaign {
	name: string;
	source: string;
	medium: string;
	term: string;
	content: string;
}

interface UserEventContextDevice {
	id: string;
	advertisingId: string;
	AdTrackingEnabled: boolean;
	manufacturer: string;
	model: string;
	name: string;
	type: string;
	token: string;
}

interface UserEventContextLibrary {
	name: string;
	version: string;
}

interface UserEventContextLocation {
	city: string;
	country: string;
	latitude: number;
	longitude: number;
	speed: number;
}

interface UserEventContextNetwork {
	bluetooth: boolean;
	carrier: string;
	cellular: boolean;
	wifi: boolean;
}

interface UserEventContextOS {
	name: string;
	version: string;
}

interface UserEventContextPage {
	path: string;
	referrer: string;
	search: string;
	title: string;
	url: string;
}

interface UserEventContextReferrer {
	id: string;
	type: string;
}

interface UserEventContextScreen {
	width: number;
	height: number;
	density: number;
}

interface UserEventContext {
	active?: boolean;
	app?: UserEventContextApp;
	browser?: UserEventContextBrowser;
	campaign?: UserEventContextCampaign;
	device?: UserEventContextDevice;
	ip?: string;
	library?: UserEventContextLibrary;
	locale?: string;
	location?: UserEventContextLocation;
	network?: UserEventContextNetwork;
	os?: UserEventContextOS;
	page?: UserEventContextPage;
	referrer?: UserEventContextReferrer;
	screen?: UserEventContextScreen;
	sessionId?: string;
	sessionStart?: boolean;
	timezone?: string;
	userAgent?: string;
}

interface UserEvent {
	anonymousId?: string;
	category?: string;
	context: UserEventContext;
	event?: string;
	groupId?: string;
	messageId?: string;
	name?: string;
	properties?: any;
	receivedAt?: string;
	sentAt?: string;
	source?: number;
	timestamp?: string;
	traits?: any;
	type?: string;
	userId?: string;
}

type UserTraits = Record<string, any>;

interface UserIdentityLabelValue {
	Label: string;
	Value: string;
}

interface UserIdentity {
	Connection: number;
	IdentityId: UserIdentityLabelValue;
	DisplayedProperty: string;
	AnonymousIds: string[] | null;
	LastChangeTime: string;
}

interface User {
	id: number;
	events: UserEvent[];
	traits: UserTraits;
}

export type { User, UserEvent, UserTraits, UserIdentity };
