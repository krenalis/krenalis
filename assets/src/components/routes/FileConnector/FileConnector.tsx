import './FileConnector.css';
import React, { useState, useLayoutEffect, useContext, useMemo } from 'react';
import appContext from '../../../context/AppContext';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { useParams, useLocation } from 'react-router-dom';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import TransformedConnection from '../../../lib/core/connection';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import ListTile from '../../base/ListTile/ListTile';
import { Link } from '../../base/Link/Link';

const FileConnector = () => {
	const [selectedStorage, setSelectedStorage] = useState<number>();

	const { setTitle, connectors, redirect, handleError, connections } = useContext(appContext);

	const params = useParams();
	const location = useLocation();

	const file = useMemo(() => {
		const name = decodeURIComponent(params.name);
		const f = connectors.find((c) => c.name === name);
		if (f == null) {
			handleError(`Connector with name ${name} doesn't exist`);
			redirect('connectors');
			return;
		}
		return f;
	}, [params.name]);

	const role = useMemo(() => {
		const r = new URL(document.location.href).searchParams.get('role');
		if (r == null || r === '') {
			return 'Source';
		} else {
			return r;
		}
	}, [location]);

	const storages = useMemo(() => {
		const s: TransformedConnection[] = [];
		for (const c of connections) {
			if (c.isFileStorage && c.role === role) {
				s.push(c);
			}
		}
		return s;
	}, [connectors]);

	useLayoutEffect(() => {
		setTitle(`Add ${file.name} file`);
	}, [file]);

	const onStorageChange = (e) => {
		setSelectedStorage(Number(e.target.value));
	};

	const onAddActionType = (target: String) => {
		const id = storages.find((s) => s.id === selectedStorage).id;
		redirect(`connections/${id}/actions/add/${target}?format=${encodeURIComponent(file.name)}`);
	};

	return (
		<div className='file-connector'>
			<div className='route-content'>
				<div className='file-connector__content'>
					{storages.length > 0 ? (
						<div className='file-connector__storage'>
							<SlSelect
								label='Storage'
								name='storages'
								value={String(selectedStorage)}
								onSlChange={onStorageChange}
							>
								{selectedStorage != null && (
									<div className='file-connector__storage-logo' slot='prefix'>
										<LittleLogo
											icon={storages.find((s) => s.id === selectedStorage).connector.icon}
										/>
									</div>
								)}
								{storages.map((s) => (
									<SlOption key={s.id} value={String(s.id)}>
										<div slot='prefix'>
											<LittleLogo icon={s.connector.icon} />
										</div>
										{s.name}
									</SlOption>
								))}
							</SlSelect>
						</div>
					) : (
						<div className='file-connector__no-storage'>
							<div>
								To add a file connection, you need to select a file storage connection to use for{' '}
								{role === 'Source' ? 'reading' : 'writing'} the file, but none are currently available.
								Please add one before proceeding.
							</div>
							<Link path={`connectors?role=${role}&category=File%20storage`}>
								<SlButton variant='neutral'>Add file storage...</SlButton>
							</Link>
						</div>
					)}
					{selectedStorage != null && (
						<div className='file-connector__action-types'>
							<ListTile
								key={'users-action-type'}
								icon={getConnectorLogo(file.icon)}
								name='Import users'
								description={`Import the users from ${file.name}`}
								className='file-connector__action-type'
								action={
									<SlButton
										size='small'
										variant='primary'
										onClick={() => {
											onAddActionType('user');
										}}
									>
										Add
									</SlButton>
								}
							/>
							{/* // TODO(Gianluca: https://github.com/meergo/meergo/issues/895
							<ListTile
								key={'groups-action-type'}
								icon={getConnectorLogo(file.icon)}
								name='Import groups'
								description={`Import the groups from ${file.name}`}
								className='file-connector__action-type'
								action={
									<SlButton
										size='small'
										variant='primary'
										onClick={() => {
											onAddActionType('group');
										}}
									>
										Add
									</SlButton>
								}
							/> */}
						</div>
					)}
				</div>
			</div>
		</div>
	);
};

export { FileConnector };
