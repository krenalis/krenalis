import './LinkedConnectionSelector.css';
import React, { ReactNode, useMemo } from 'react';
import { useLinkedConnectionsGrid } from './useLinkedConnectionsGrid';
import TransformedConnection, { isEventConnection } from '../../../lib/core/connection';
import { ConnectionRole } from '../../../lib/api/types/connection';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import LittleLogo from '../LittleLogo/LittleLogo';
import Grid from '../Grid/Grid';

interface ConnectionSelectorProps {
	linkedConnections: Number[] | null;
	setLinkedConnections: React.Dispatch<React.SetStateAction<Number[] | null>>;
	connections: TransformedConnection[];
	role: ConnectionRole;
	title?: ReactNode;
	description?: ReactNode;
	onLink?: (id: Number) => Promise<void>;
	onUnlink?: (id: Number) => Promise<void>;
	isClickable?: boolean;
	children?: ReactNode;
}

const LinkedConnectionSelector = ({
	linkedConnections,
	setLinkedConnections,
	connections,
	role,
	title,
	description,
	onLink,
	onUnlink,
	isClickable,
}: ConnectionSelectorProps) => {
	const { rows, columns } = useLinkedConnectionsGrid(linkedConnections, setLinkedConnections, onUnlink, isClickable);

	const { linkableConnections, selectableConnections } = useMemo(() => {
		const linkableConnections: TransformedConnection[] = [];
		const selectableConnections: TransformedConnection[] = [];
		for (const c of connections) {
			if (isEventConnection(c.role, c.connector.type, c.connector.targets) && c.role !== role) {
				linkableConnections.push(c);
				const isAlreadySelected = linkedConnections?.find((id) => id === c.id) != null;
				if (!isAlreadySelected) {
					selectableConnections.push(c);
				}
			}
		}
		return { linkableConnections, selectableConnections };
	}, [connections, linkedConnections]);

	const onSelectLinkedConnection = async (e) => {
		const id = Number(e.detail.item.value);
		let updated: Number[] = [];
		if (linkedConnections != null) {
			updated = [...linkedConnections];
		}
		updated.push(id);
		if (onLink) {
			try {
				await onLink(id);
			} catch (err) {
				return;
			}
		}
		setLinkedConnections(updated);
	};

	const hasSelectableConnections = selectableConnections.length > 0;

	return (
		<div className='linked-connection-selector'>
			{linkableConnections.length === 0 ? (
				`Currently there is no linkable ${role === 'Source' ? 'destination' : 'source'}`
			) : (
				<div className={'linked-connection-selector__content'}>
					{(title != null || description != null || hasSelectableConnections) && (
						<div
							className={`linked-connection-selector__head${
								(title || description) && hasSelectableConnections
									? ' linked-connection-selector__head--flex'
									: ''
							}`}
						>
							<div className='linked-connection-selector__head-text'>
								<div className='linked-connection-selector__head-title'>{title}</div>
								<div className='linked-connection-selector__head-description'>{description}</div>
							</div>
							{hasSelectableConnections && (
								<SlDropdown className='linked-connection-selector__dropdown'>
									<SlButton slot='trigger' caret>
										<SlIcon slot='prefix' name='plus' />
										Link {role === 'Source' ? 'destination' : 'source'}...
									</SlButton>
									<SlMenu onSlSelect={onSelectLinkedConnection}>
										{selectableConnections.map((c) => {
											const isAlreadySelected =
												linkedConnections?.find((id) => id === c.id) != null;
											if (!isAlreadySelected) {
												return (
													<SlMenuItem key={c.id} value={String(c.id)}>
														<LittleLogo icon={c.connector.icon} />
														{c.name}
													</SlMenuItem>
												);
											}
										})}
									</SlMenu>
								</SlDropdown>
							)}
						</div>
					)}
					<Grid
						rows={rows}
						columns={columns}
						noRowsMessage={`Select "Link ${role === 'Source' ? 'destination' : 'source'}..." to link a ${
							role === 'Source' ? 'destination' : 'source'
						} connection`}
					/>
					{!hasSelectableConnections && (
						<div className='linked-connection-selector__no-connection-message'>
							There are no other event {role === 'Source' ? 'destinations' : 'sources'} available to be
							linked, beyond those that have already been linked.
						</div>
					)}
				</div>
			)}
		</div>
	);
};

export { LinkedConnectionSelector };
