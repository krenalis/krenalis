import './EventConnectionSelector.css';
import React, { ReactNode, useMemo } from 'react';
import { useEventConnectionsGrid } from './useEventConnectionsGrid';
import TransformedConnection, { isEventConnection } from '../../../lib/helpers/transformedConnection';
import { ConnectionRole } from '../../../lib/api/types/connection';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlMenuItem from '@shoelace-style/shoelace/dist/react/menu-item/index.js';
import LittleLogo from '../../shared/LittleLogo/LittleLogo';
import Grid from '../Grid/Grid';

interface ConnectionSelectorProps {
	eventConnections: Number[] | null;
	setEventConnections: React.Dispatch<React.SetStateAction<Number[] | null>>;
	connections: TransformedConnection[];
	role: ConnectionRole;
	title?: ReactNode;
	description?: string;
	onAdd?: (id: Number) => Promise<void>;
	onRemove?: (id: Number) => Promise<void>;
	isClickable?: boolean;
	isShown?: boolean;
	children?: ReactNode;
}

const EventConnectionSelector = ({
	eventConnections,
	setEventConnections,
	connections,
	role,
	title,
	description,
	onAdd,
	onRemove,
	isClickable,
	isShown,
}: ConnectionSelectorProps) => {
	const { rows, columns } = useEventConnectionsGrid(eventConnections, setEventConnections, onRemove, isClickable);

	const { linkableConnections, selectableConnections } = useMemo(() => {
		const linkableConnections: TransformedConnection[] = [];
		const selectableConnections: TransformedConnection[] = [];
		for (const c of connections) {
			if (isEventConnection(c.role, c.type, c.connector.targets) && c.role !== role) {
				linkableConnections.push(c);
				const isAlreadySelected = eventConnections?.find((id) => id === c.id) != null;
				if (!isAlreadySelected) {
					selectableConnections.push(c);
				}
			}
		}
		return { linkableConnections, selectableConnections };
	}, [connections, eventConnections]);

	const onSelectEventConnection = async (e) => {
		const id = Number(e.detail.item.value);
		let updated: Number[] = [];
		if (eventConnections != null) {
			updated = [...eventConnections];
		}
		updated.push(id);
		if (onAdd) {
			try {
				await onAdd(id);
			} catch (err) {
				return;
			}
		}
		setEventConnections(updated);
	};

	const hasSelectableConnections = selectableConnections.length > 0;

	return (
		<div className='event-connection-selector'>
			{linkableConnections.length === 0 ? (
				`Currently there is no event ${role === 'Source' ? 'destination' : 'source'}`
			) : (
				<div className={'event-connection-selector__content'}>
					<div
						className={`event-connection-selector__head${
							(title || description) && hasSelectableConnections
								? ' event-connection-selector__head--flex'
								: ''
						}`}
					>
						<div className='event-connection-selector__head-text'>
							{title}
							<div className='event-connection-selector__head-description'>{description}</div>
						</div>
						{hasSelectableConnections && (
							<SlDropdown className='event-connection-selector__dropdown'>
								<SlButton slot='trigger' caret>
									<SlIcon slot='prefix' name='plus' />
									Add {role === 'Source' ? 'destination' : 'source'}...
								</SlButton>
								<SlMenu onSlSelect={onSelectEventConnection}>
									{selectableConnections.map((c) => {
										const isAlreadySelected = eventConnections?.find((id) => id === c.id) != null;
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
					<Grid
						rows={rows}
						columns={columns}
						noRowsMessage={`Select "Add ${role === 'Source' ? 'destination' : 'source'}..." to add a ${
							role === 'Source' ? 'destination' : 'source'
						} connection`}
						isShown={isShown}
					/>
					{!hasSelectableConnections && (
						<div className='event-connection-selector__no-connection-message'>
							There are no other event {role === 'Source' ? 'destinations' : 'sources'} available to be
							added, beyond those that have already been added.
						</div>
					)}
				</div>
			)}
		</div>
	);
};

export { EventConnectionSelector };
