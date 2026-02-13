import React from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import './DocumentationLinks.css';

// Mapping: connectorCode -> role -> links
const DOCUMENTATION_LINKS: Record<string, Record<string, { label: string; url: string }[]>> = {
	// Databases
	postgresql: {
		Source: [
			{
				label: 'Ingest users from PostgreSQL',
				url: 'https://www.meergo.com/docs/ingest-users/databases?settings=postgresql#1-connect-a-database',
			},
		],
		Destination: [
			{
				label: 'Activate profiles to PostgreSQL',
				url: 'https://www.meergo.com/docs/activate-profiles/databases?settings=postgresql#1-connect-a-database-table',
			},
		],
	},
	mysql: {
		Source: [
			{
				label: 'Ingest users from MySQL',
				url: 'https://www.meergo.com/docs/ingest-users/databases?settings=mysql#1-connect-a-database',
			},
		],
		Destination: [
			{
				label: 'Activate profiles to MySQL',
				url: 'https://www.meergo.com/docs/activate-profiles/databases?settings=mysql#1-connect-a-database-table',
			},
		],
	},
	snowflake: {
		Source: [
			{
				label: 'Ingest users from Snowflake',
				url: 'https://www.meergo.com/docs/ingest-users/databases?settings=snowflake#1-connect-a-database',
			},
		],
		Destination: [
			{
				label: 'Activate profiles to Snowflake',
				url: 'https://www.meergo.com/docs/activate-profiles/databases?settings=snowflake#1-connect-a-database-table',
			},
		],
	},
	clickhouse: {
		Source: [
			{
				label: 'Ingest users from ClickHouse',
				url: 'https://www.meergo.com/docs/ingest-users/databases?settings=clickhouse#1-connect-a-database',
			},
		],
		Destination: [
			{
				label: 'Activate profiles to ClickHouse',
				url: 'https://www.meergo.com/docs/activate-profiles/databases?settings=clickhouse#1-connect-a-database-table',
			},
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
			{
				label: 'Ingest users from Mailchimp',
				url: 'https://www.meergo.com/docs/ingest-users/saas-apps/mailchimp',
			},
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
			{
				label: 'Activate events on Google Analytics',
				url: 'https://www.meergo.com/docs/activate-events/google-analytics',
			},
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
			{
				label: 'Ingest users from RudderStack',
				url: 'https://www.meergo.com/docs/ingest-users/saas-apps/rudderstack',
			},
		],
	},
	// File storages
	s3: {
		Source: [
			{
				label: 'Ingest users from files on S3',
				url: 'https://www.meergo.com/docs/ingest-users/files?storage=s3#1-connect-a-storage',
			},
		],
		Destination: [
			{
				label: 'Activate profiles to files on S3',
				url: 'https://www.meergo.com/docs/activate-profiles/files?storage=s3#1-connect-a-storage',
			},
		],
	},
	sftp: {
		Source: [
			{
				label: 'Ingest users from files on SFTP',
				url: 'https://www.meergo.com/docs/ingest-users/files?storage=sftp#1-connect-a-storage',
			},
		],
		Destination: [
			{
				label: 'Activate profiles to files on SFTP',
				url: 'https://www.meergo.com/docs/activate-profiles/files?storage=sftp#1-connect-a-storage',
			},
		],
	},
	filesystem: {
		Source: [
			{
				label: 'Ingest users from files on File System',
				url: 'https://www.meergo.com/docs/ingest-users/files?storage=filesystem#1-connect-a-storage',
			},
		],
		Destination: [
			{
				label: 'Activate profiles to files on File System',
				url: 'https://www.meergo.com/docs/activate-profiles/files?storage=filesystem#1-connect-a-storage',
			},
		],
	},
	'http-get': {
		Source: [
			{
				label: 'Ingest users from files via HTTP GET',
				url: 'https://www.meergo.com/docs/ingest-users/files?storage=http#panel-storage-http-get',
			},
		],
	},
	'http-post': {
		Destination: [
			{
				label: 'Activate profiles to files via HTTP POST',
				url: 'https://www.meergo.com/docs/activate-profiles/files?storage=http#panel-storage-http-post',
			},
		],
	},
	// SDKs
	javascript: {
		Source: [
			{
				label: 'Collect events with JavaScript SDK',
				url: 'https://www.meergo.com/docs/collect-events/apps-you-developed?sdk=javascript#1-connect-an-application',
			},
			{
				label: 'Ingest users with JavaScript SDK',
				url: 'https://www.meergo.com/docs/ingest-users/apps-you-developed?sdk=javascript#1-connect-an-application',
			},
		],
	},
	android: {
		Source: [
			{
				label: 'Collect events with Android SDK',
				url: 'https://www.meergo.com/docs/collect-events/apps-you-developed?sdk=android#1-connect-an-application',
			},
			{
				label: 'Ingest users with Android SDK',
				url: 'https://www.meergo.com/docs/ingest-users/apps-you-developed?sdk=android#1-connect-an-application',
			},
		],
	},
	nodejs: {
		Source: [
			{
				label: 'Collect events with Node.js SDK',
				url: 'https://www.meergo.com/docs/collect-events/apps-you-developed?sdk=nodejs#1-connect-an-application',
			},
			{
				label: 'Ingest users with Node.js SDK',
				url: 'https://www.meergo.com/docs/ingest-users/apps-you-developed?sdk=nodejs#1-connect-an-application',
			},
		],
	},
	python: {
		Source: [
			{
				label: 'Collect events with Python SDK',
				url: 'https://www.meergo.com/docs/collect-events/apps-you-developed?sdk=python#1-connect-an-application',
			},
			{
				label: 'Ingest users with Python SDK',
				url: 'https://www.meergo.com/docs/ingest-users/apps-you-developed?sdk=python#1-connect-an-application',
			},
		],
	},
	go: {
		Source: [
			{
				label: 'Collect events with Go SDK',
				url: 'https://www.meergo.com/docs/collect-events/apps-you-developed?sdk=go#1-connect-an-application',
			},
			{
				label: 'Ingest users with Go SDK',
				url: 'https://www.meergo.com/docs/ingest-users/apps-you-developed?sdk=go#1-connect-an-application',
			},
		],
	},
	java: {
		Source: [
			{
				label: 'Collect events with Java SDK',
				url: 'https://www.meergo.com/docs/collect-events/apps-you-developed?sdk=java#1-connect-an-application',
			},
			{
				label: 'Ingest users with Java SDK',
				url: 'https://www.meergo.com/docs/ingest-users/apps-you-developed?sdk=java#1-connect-an-application',
			},
		],
	},
	dotnet: {
		Source: [
			{
				label: 'Collect events with .NET SDK',
				url: 'https://www.meergo.com/docs/collect-events/apps-you-developed?sdk=net#1-connect-an-application',
			},
			{
				label: 'Ingest users with .NET SDK',
				url: 'https://www.meergo.com/docs/ingest-users/apps-you-developed?sdk=net#1-connect-an-application',
			},
		],
	},
};

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
		const basePath = isSource ? 'ingest-users' : 'activate-profiles';
		const fragment = isSource ? '4-enter-file-settings' : '5-enter-file-settings';
		const url = `https://www.meergo.com/docs/${basePath}/files?storage=${storageCode}&format=${connectorCode}#${fragment}`;
		links = [{ label, url }];
	} else {
		links = DOCUMENTATION_LINKS[connectorCode]?.[role];
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
