import React from 'react';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import WaDropdown from '@awesome.me/webawesome/dist/react/dropdown/index.js';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
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
		<WaDropdown
			className={`pipeline__issues${show && count > 0 ? ' pipeline__issues--visible' : ''}`}
			distance={10}
			slot={slot}
			placement='bottom'
		>
			<WaButton variant='warning' slot='trigger' withCaret>
				<WaIcon slot='start' name='exclamation-triangle' />
				{`${count === 1 ? '1 issue' : count + ' issues'} with the ${labelTarget}`}
			</WaButton>
			<div className='pipeline__issues-list'>
				{issues.map((issue) => {
					return <div className='pipeline__issue'>{issue}</div>;
				})}
			</div>
		</WaDropdown>
	);
};

export { PipelineIssues };
