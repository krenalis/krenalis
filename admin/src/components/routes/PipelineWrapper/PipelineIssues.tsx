import React from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { ConnectionRole, ConnectorType } from '../../../lib/api/types/connection';

interface PipelineIssuesProps {
	issues: string[];
	type: ConnectorType;
	role: ConnectionRole;
	show?: boolean;
	slot?: string;
}

const PipelineIssues = ({ issues, type, role, show = true, slot }: PipelineIssuesProps) => {
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
			className={`pipeline__issues${show && count > 0 ? ' pipeline__issues--visible' : ''}`}
			distance={10}
			slot={slot}
			placement='bottom'
			hoist
		>
			<SlButton variant='warning' slot='trigger' caret>
				<SlIcon slot='prefix' name='exclamation-triangle' />
				{`${count === 1 ? '1 issue' : count + ' issues'} with the ${labelTarget}`}
			</SlButton>
			<div className='pipeline__issues-list'>
				{issues.map((issue) => {
					return <div className='pipeline__issue'>{issue}</div>;
				})}
			</div>
		</SlDropdown>
	);
};

export { PipelineIssues };
