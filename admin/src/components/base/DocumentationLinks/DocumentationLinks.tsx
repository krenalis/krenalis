import React from 'react';
import './DocumentationLinks.css';

// Mapping: connectorCode -> role -> links
const DOCUMENTATION_LINKS: Record<string, Record<string, { label: string; url: string }[]>> = {
	klaviyo: {
		Destination: [
			{
				label: 'Activate profiles on Klaviyo',
				url: 'https://www.meergo.com/docs/activate-profiles/klaviyo',
			},
			{
				label: 'Activate events on Klaviyo',
				url: 'https://www.meergo.com/docs/activate-events/klaviyo',
			},
		],
	},
	'http-get': {
		Source: [
			{
				label: 'Learn how to ingest users from HTTP GET',
				url: 'https://www.meergo.com/docs/ingest-users/files?storage=http#panel-storage-http-get',
			},
		],
	},
	postgresql: {
		Source: [{ label: 'How to configure PostgreSQL as a source', url: 'https://docs.example.com/postgres-source' }],
		Destination: [{ label: 'PostgreSQL destination setup', url: 'https://docs.example.com/postgres-dest' }],
	},
};

interface DocumentationLinksProps {
	connectorCode: string;
	role: string;
}

const DocumentationLinks = ({ connectorCode, role }: DocumentationLinksProps) => {
	const links = DOCUMENTATION_LINKS[connectorCode]?.[role];
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
