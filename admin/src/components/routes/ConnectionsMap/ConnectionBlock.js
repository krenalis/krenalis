import { useContext } from 'react';
import Flex from '../../common/Flex/Flex';
import StatusDot from '../../common/StatusDot/StatusDot';
import { AppContext } from '../../../providers/AppProvider';
import { SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionBlock = ({ connection: c, isNew }) => {
	const { redirect } = useContext(AppContext);

	const onClick = () => {
		redirect(`connections/${c.id}/actions`);
	};

	return (
		<div className={`connectionBlock${isNew ? ' new' : ''}`} id={`${c.id}`} onClick={onClick}>
			<Flex alignItems='center' justifyContent='space-between' gap={20}>
				<Flex alignItems='center' gap={10}>
					{c.logo}
					<div className='name'>{c.name}</div>
				</Flex>
				<SlTooltip content={c.status.text}>
					<StatusDot status={c.status} />
				</SlTooltip>
			</Flex>
		</div>
	);
};

export default ConnectionBlock;
