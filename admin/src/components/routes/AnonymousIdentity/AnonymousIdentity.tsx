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
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import { ObjectType } from '../../../types/external/types';

const AnonymousIdentity = () => {
	const [anonymousIdentifiers, setAnonymousIdentifiers] = useState<TransformedIdentifiers>();
	const [eventSchema, setEventSchema] = useState<ObjectType>();
	const [userSchema, setUserSchema] = useState<ObjectType>();
	const [isLoading, setIsLoading] = useState<boolean>(true);

	const { setTitle, api, showError, showStatus, workspace, setIsWorkspaceStale } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Anonymous IDs');
	}, []);

	useEffect(() => {
		const fetchData = async () => {
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
				userSchema = await api.workspace.userSchema();
			} catch (err) {
				showError(err);
				return;
			}
			setUserSchema(userSchema);
			setIsLoading(false);
		};
		fetchData();
	}, []);

	const onSave = async () => {
		const errorMessage = validateIdentifiersMapping(anonymousIdentifiers!);
		if (errorMessage) {
			showError(errorMessage);
			return;
		}
		const untransformed = untransformAnonymousIdentifiers(anonymousIdentifiers!);
		try {
			await api.workspace.anonymousIdentifiers(untransformed);
		} catch (err) {
			showError(err);
			return;
		}
		showStatus({ variant: variants.SUCCESS, icon: icons.OK, text: 'Anonymous identifiers saved succesfully' });
		setIsWorkspaceStale(true);
	};

	return (
		<div className='anonymousIdentity'>
			{isLoading ? (
				<SlSpinner
					style={
						{
							fontSize: '3rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				></SlSpinner>
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
					<SlButton className='saveButton' onClick={onSave} variant='primary'>
						Save
					</SlButton>
				</Section>
			)}
		</div>
	);
};

export default AnonymousIdentity;
