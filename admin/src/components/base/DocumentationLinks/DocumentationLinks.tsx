import React from 'react';
import './DocumentationLinks.css';

// Mapping: connectorCode -> role -> links
const DOCUMENTATION_LINKS: Record<string, Record<string, { label: string; url: string }[]>> = {
	// Databases
	postgresql: {
		Source: [
			{ label: 'Ingest users from PostgreSQL', url: 'https://www.meergo.com/docs/ingest-users/databases' },
		],
		Destination: [
			{ label: 'Activate profiles to PostgreSQL', url: 'https://www.meergo.com/docs/activate-profiles/databases' },
		],
	},
	mysql: {
		Source: [
			{ label: 'Ingest users from MySQL', url: 'https://www.meergo.com/docs/ingest-users/databases' },
		],
		Destination: [
			{ label: 'Activate profiles to MySQL', url: 'https://www.meergo.com/docs/activate-profiles/databases' },
		],
	},
	snowflake: {
		Source: [
			{ label: 'Ingest users from Snowflake', url: 'https://www.meergo.com/docs/ingest-users/databases' },
		],
		Destination: [
			{ label: 'Activate profiles to Snowflake', url: 'https://www.meergo.com/docs/activate-profiles/databases' },
		],
	},
	clickhouse: {
		Source: [
			{ label: 'Ingest users from ClickHouse', url: 'https://www.meergo.com/docs/ingest-users/databases' },
		],
		Destination: [
			{ label: 'Activate profiles to ClickHouse', url: 'https://www.meergo.com/docs/activate-profiles/databases' },
		],
	},
	// SaaS apps (Source + Destination)
	stripe: {
		Source: [
			{ label: 'Ingest users from Stripe', url: 'https://www.meergo.com/docs/ingest-users/saas-apps/stripe' },
		],
		Destination: [
			{ label: 'Activate profiles on Stripe', url: 'https://www.meergo.com/docs/activate-profiles/stripe' },
		],
	},
	hubspot: {
		Source: [
			{ label: 'Ingest users from HubSpot', url: 'https://www.meergo.com/docs/ingest-users/saas-apps/hubspot' },
		],
		Destination: [
			{ label: 'Activate profiles on HubSpot', url: 'https://www.meergo.com/docs/activate-profiles/hubspot' },
		],
	},
	klaviyo: {
		Source: [
			{ label: 'Ingest users from Klaviyo', url: 'https://www.meergo.com/docs/ingest-users/saas-apps/klaviyo' },
		],
		Destination: [
			{ label: 'Activate profiles on Klaviyo', url: 'https://www.meergo.com/docs/activate-profiles/klaviyo' },
			{ label: 'Activate events on Klaviyo', url: 'https://www.meergo.com/docs/activate-events/klaviyo' },
		],
	},
	mailchimp: {
		Source: [
			{ label: 'Ingest users from Mailchimp', url: 'https://www.meergo.com/docs/ingest-users/saas-apps/mailchimp' },
		],
		Destination: [
			{ label: 'Activate profiles on Mailchimp', url: 'https://www.meergo.com/docs/activate-profiles/mailchimp' },
		],
	},
	// SaaS apps (Destination only, events)
	mixpanel: {
		Destination: [
			{ label: 'Activate events on Mixpanel', url: 'https://www.meergo.com/docs/activate-events/mixpanel' },
		],
	},
	'google-analytics': {
		Destination: [
			{ label: 'Activate events on Google Analytics', url: 'https://www.meergo.com/docs/activate-events/google-analytics' },
		],
	},
	posthog: {
		Destination: [
			{ label: 'Activate events on PostHog', url: 'https://www.meergo.com/docs/activate-events/posthog' },
		],
	},
	// SaaS apps (Source only)
	segment: {
		Source: [
			{ label: 'Ingest users from Segment', url: 'https://www.meergo.com/docs/ingest-users/saas-apps/segment' },
		],
	},
	rudderstack: {
		Source: [
			{ label: 'Ingest users from RudderStack', url: 'https://www.meergo.com/docs/ingest-users/saas-apps/rudderstack' },
		],
	},
	// File storages
	s3: {
		Source: [
			{ label: 'Ingest users from files on S3', url: 'https://www.meergo.com/docs/ingest-users/files?storage=s3' },
		],
		Destination: [
			{ label: 'Activate profiles to files on S3', url: 'https://www.meergo.com/docs/activate-profiles/files?storage=s3' },
		],
	},
	sftp: {
		Source: [
			{ label: 'Ingest users from files on SFTP', url: 'https://www.meergo.com/docs/ingest-users/files?storage=sftp' },
		],
		Destination: [
			{ label: 'Activate profiles to files on SFTP', url: 'https://www.meergo.com/docs/activate-profiles/files?storage=sftp' },
		],
	},
	filesystem: {
		Source: [
			{ label: 'Ingest users from files on File System', url: 'https://www.meergo.com/docs/ingest-users/files?storage=filesystem' },
		],
		Destination: [
			{ label: 'Activate profiles to files on File System', url: 'https://www.meergo.com/docs/activate-profiles/files?storage=filesystem' },
		],
	},
	'http-get': {
		Source: [
			{ label: 'Ingest users from files via HTTP GET', url: 'https://www.meergo.com/docs/ingest-users/files?storage=http#panel-storage-http-get' },
		],
	},
	'http-post': {
		Destination: [
			{ label: 'Activate profiles to files via HTTP POST', url: 'https://www.meergo.com/docs/activate-profiles/files?storage=http#panel-storage-http-post' },
		],
	},
	// SDKs
	javascript: {
		Source: [
			{ label: 'JavaScript SDK documentation', url: 'https://www.meergo.com/docs/integrations/javascript-sdk' },
		],
	},
	android: {
		Source: [
			{ label: 'Android SDK documentation', url: 'https://www.meergo.com/docs/integrations/android-sdk' },
		],
	},
	nodejs: {
		Source: [
			{ label: 'Node.js SDK documentation', url: 'https://www.meergo.com/docs/integrations/nodejs-sdk' },
		],
	},
	python: {
		Source: [
			{ label: 'Python SDK documentation', url: 'https://www.meergo.com/docs/integrations/python-sdk' },
		],
	},
	go: {
		Source: [
			{ label: 'Go SDK documentation', url: 'https://www.meergo.com/docs/integrations/go-sdk' },
		],
	},
	java: {
		Source: [
			{ label: 'Java SDK documentation', url: 'https://www.meergo.com/docs/integrations/java-sdk' },
		],
	},
	dotnet: {
		Source: [
			{ label: '.NET SDK documentation', url: 'https://www.meergo.com/docs/integrations/dotnet-sdk' },
		],
	},
};

interface DocumentationLinksProps {
	connectorCode: string;
	role: string;
	storageCode?: string;
	connectorLabel?: string;
}

const DocumentationLinks = ({ connectorCode, role, storageCode, connectorLabel }: DocumentationLinksProps) => {
	let links: { label: string; url: string }[];

	if (storageCode != null && connectorLabel != null) {
		const isSource = role === 'Source';
		const label = isSource ? `Import users from ${connectorLabel}` : `Export users to ${connectorLabel}`;
		const basePath = isSource ? 'ingest-users' : 'activate-users';
		const url = `https://www.meergo.com/docs/${basePath}/files?storage=${storageCode}&format=${connectorCode}`;
		links = [{ label, url }];
	} else {
		links = DOCUMENTATION_LINKS[connectorCode]?.[role];
	}

	if (!links || links.length === 0) return null;

	return (
		<div className='documentation-links'>
			{links.map((link) => (
				<a key={link.url} href={link.url} target='_blank' rel='noopener'>
					{link.label}
				</a>
			))}
		</div>
	);
};

export default DocumentationLinks;
