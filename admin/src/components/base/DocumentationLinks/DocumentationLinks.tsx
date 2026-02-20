import React from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import './DocumentationLinks.css';

const CONNECTOR_DISPLAY_NAMES: Record<string, string> = {
	postgresql: 'PostgreSQL',
	mysql: 'MySQL',
	snowflake: 'Snowflake',
	clickhouse: 'ClickHouse',
	stripe: 'Stripe',
	hubspot: 'HubSpot',
	klaviyo: 'Klaviyo',
	mailchimp: 'Mailchimp',
	mixpanel: 'Mixpanel',
	'google-analytics': 'Google Analytics',
	posthog: 'PostHog',
	segment: 'Segment',
	rudderstack: 'RudderStack',
	s3: 'files on S3',
	sftp: 'files on SFTP',
	filesystem: 'files on File System',
	'http-get': 'files via HTTP GET',
	'http-post': 'files via HTTP POST',
	javascript: 'JavaScript SDK',
	android: 'Android SDK',
	nodejs: 'Node.js SDK',
	python: 'Python SDK',
	go: 'Go SDK',
	java: 'Java SDK',
	dotnet: '.NET SDK',
};

const STORAGE_URL_SLUGS: Record<string, string> = {
	s3: 's3',
	sftp: 'sftp',
	filesystem: 'file-system',
	'http-get': 'http-get',
};

const SDK_CONNECTORS = new Set(['javascript', 'android', 'nodejs', 'python', 'go', 'java', 'dotnet']);
const EVENTS_ONLY_DESTINATIONS = new Set(['mixpanel', 'google-analytics', 'posthog']);
const USERS_AND_EVENTS_DESTINATIONS = new Set(['klaviyo']);
const PREPOSITION_TO = new Set([
	'postgresql',
	'mysql',
	'snowflake',
	'clickhouse',
	's3',
	'sftp',
	'filesystem',
	'http-post',
]);

// getConnectionDocLinks returns the documentation links for a given connector
// and role. Connectors without a declared display name in
// CONNECTOR_DISPLAY_NAMES are intentionally excluded (returns an empty array),
// since their docs page does not exist.
function getConnectionDocLinks(connectorCode: string, role: string): { label: string; url: string }[] {
	const isSource = role === 'Source';
	const direction = isSource ? 'sources' : 'destinations';
	const baseUrl = `https://www.meergo.com/docs/ref/admin/connection-configuration/${direction}-${connectorCode}`;
	const name = CONNECTOR_DISPLAY_NAMES[connectorCode];
	if (name == null) return [];

	if (SDK_CONNECTORS.has(connectorCode)) {
		return [
			{ label: `Collect events with ${name}`, url: `${baseUrl}-events` },
			{ label: `Ingest users with ${name}`, url: `${baseUrl}-users` },
		];
	}

	const prep = PREPOSITION_TO.has(connectorCode) ? 'to' : 'on';
	const links: { label: string; url: string }[] = [];

	if (isSource) {
		links.push({ label: `Ingest users from ${name}`, url: `${baseUrl}-users` });
	} else {
		if (!EVENTS_ONLY_DESTINATIONS.has(connectorCode)) {
			links.push({ label: `Activate profiles ${prep} ${name}`, url: `${baseUrl}-users` });
		}
		if (EVENTS_ONLY_DESTINATIONS.has(connectorCode) || USERS_AND_EVENTS_DESTINATIONS.has(connectorCode)) {
			links.push({ label: `Activate events on ${name}`, url: `${baseUrl}-events` });
		}
	}

	return links;
}

interface DocumentationLinksProps {
	connectorCode: string;
	role: string;
	storageCode?: string;
	connectorLabel?: string;
	fade?: boolean;
	showIcon?: boolean;
}

const DocumentationLinks = ({
	connectorCode,
	role,
	storageCode,
	connectorLabel,
	fade = false,
	showIcon = false,
}: DocumentationLinksProps) => {
	let links: { label: string; url: string }[];

	if (storageCode != null && connectorLabel != null) {
		const isSource = role === 'Source';
		const label = isSource
			? `How to import users from ${connectorLabel}`
			: `How to export users to ${connectorLabel}`;
		const direction = isSource ? 'sources' : 'destinations';
		const storageSlug = STORAGE_URL_SLUGS[storageCode] ?? storageCode;
		const url = `https://www.meergo.com/docs/ref/admin/pipeline-configuration/${direction}-${connectorCode}-on-${storageSlug}`;
		links = [{ label, url }];
	} else {
		links = getConnectionDocLinks(connectorCode, role);
	}

	if (!links || links.length === 0) return null;

	return (
		<div className={`documentation-links${fade ? ' documentation-links--fade' : ''}`}>
			{links.map((link) => (
				<a key={link.url} href={link.url} target='_blank' rel='noopener'>
					{showIcon && <SlIcon name='question-circle' />}
					{link.label}
				</a>
			))}
		</div>
	);
};

export default DocumentationLinks;
