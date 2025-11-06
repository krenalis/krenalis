import React from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { ConnectionRole, ConnectorType } from '../../../lib/api/types/connection';

interface ActionIssuesProps {
	issues: string[];
	type: ConnectorType;
	role: ConnectionRole;
	show?: boolean;
	slot?: string;
}

const ActionIssues = ({ issues, type, role, show = true, slot }: ActionIssuesProps) => {
	let count = issues.length;

	let labelTarget = '';
	if (type === 'FileStorage') {
		labelTarget = 'file';
	} else if (type === 'Database') {
		if (role === 'Source') {
			labelTarget = 'query';
		} else {
			labelTarget = 'table';
		}
	}

	return (
		<SlDropdown
			className={`action__issues${show && count > 0 ? ' action__issues--visible' : ''}`}
			distance={10}
			slot={slot}
			placement='bottom'
			hoist
		>
			<SlButton variant='warning' slot='trigger' caret>
				<SlIcon slot='prefix' name='exclamation-triangle' />
				{`${count === 1 ? '1 issue' : count + ' issues'} with the ${labelTarget}`}
			</SlButton>
			<div className='action__issues-list'>
				{issues.map((issue) => {
					return <div className='action__issue'>{issue}</div>;
				})}
			</div>
		</SlDropdown>
	);
};

export { ActionIssues };
