import React, { useEffect, useLayoutEffect, useState } from 'react';
import './AnonymousIdentity.css';
import IdentifiersMapping from '../../shared/IdentifiersMapping/IdentifiersMapping';
import Section from '../../shared/Section/Section';
import * as variants from '../../../constants/variants';
import * as icons from '../../../constants/icons';
import { useContext } from 'react';
import { AppContext } from '../../../context/providers/AppProvider';
import {
	validateIdentifiersMapping,
	transformAnonymousIdentifiers,
	untransformAnonymousIdentifiers,
	TransformedIdentifiers,
} from '../../../lib/helpers/transformedIdentifiers';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import { ObjectType } from '../../../types/external/types';

const AnonymousIdentity = () => {
	const [anonymousIdentifiers, setAnonymousIdentifiers] = useState<TransformedIdentifiers>();
	const [eventSchema, setEventSchema] = useState<ObjectType>();
	const [userSchema, setUserSchema] = useState<ObjectType>();
	const [isLoading, setIsLoading] = useState<boolean>(true);

	const { api, showError, showStatus, workspaces, selectedWorkspace, setIsWorkspaceStale, redirect } =
		useContext(AppContext);

	useEffect(() => {
		const fetchData = async () => {
			const workspace = workspaces.find((w) => w.ID === selectedWorkspace);
			const transformed = transformAnonymousIdentifiers(workspace.AnonymousIdentifiers);
			setAnonymousIdentifiers(transformed);

			let eventSchema: ObjectType;
			try {
				eventSchema = await api.eventsSchema();
			} catch (err) {
				showError(err);
				return;
			}
			setEventSchema(eventSchema);

			let userSchema: ObjectType;
			try {
				userSchema = await api.workspaces.userSchema();
			} catch (err) {
				showError(err);
				return;
			}
			setUserSchema(userSchema);
			setIsLoading(false);
		};
		fetchData();
	}, [selectedWorkspace]);

	const onConnectDataWarehouse = () => {
		redirect('settings/data-warehouse');
	};

	const onSave = async () => {
		const errorMessage = validateIdentifiersMapping(anonymousIdentifiers!);
		if (errorMessage) {
			showError(errorMessage);
			return;
		}
		const untransformed = untransformAnonymousIdentifiers(anonymousIdentifiers!);
		try {
			await api.workspaces.anonymousIdentifiers(untransformed);
		} catch (err) {
			showError(err);
			return;
		}
		showStatus({ variant: variants.SUCCESS, icon: icons.OK, text: 'Anonymous identifiers saved succesfully' });
		setIsWorkspaceStale(true);
	};

	return (
		<div className='anonymous-identity'>
			{isLoading ? (
				<SlSpinner
					style={
						{
							fontSize: '3rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				></SlSpinner>
			) : userSchema == null ? (
				<div className='anonymous-identity__no-schema'>
					<div className='anonymous-identity__no-schema-title'>Connect a data warehouse</div>
					<div className='anonymous-identity__no-schema-description'>
						The anonymous identifiers are chosen from among the data warehouse columns. For this reason,
						before you can set the anonymous identifiers you must connect a data warehouse.
					</div>
					<SlButton
						variant='primary'
						className='anonymous-identity__connect-warehouse-button'
						onClick={onConnectDataWarehouse}
					>
						<SlIcon name='database' slot='prefix' />
						Connect a data warehouse...
					</SlButton>
				</div>
			) : (
				<Section
					title='Anonymous Identifiers'
					description='Define the identifiers used to resolve the identity of anonymous users'
				>
					<IdentifiersMapping
						mapping={anonymousIdentifiers!}
						setMapping={setAnonymousIdentifiers}
						inputSchema={eventSchema!}
						outputSchema={userSchema!}
					/>
					<SlButton className='anonymous-identity__save-button' onClick={onSave} variant='primary'>
						Save
					</SlButton>
				</Section>
			)}
		</div>
	);
};

export default AnonymousIdentity;
