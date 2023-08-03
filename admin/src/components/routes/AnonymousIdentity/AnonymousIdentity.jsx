import { useEffect, useLayoutEffect, useState } from 'react';
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
} from '../../../lib/helpers/identifiers';
import { SlButton, SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

const AnonymousIdentity = () => {
	const [anonymousIdentifiers, setAnonymousIdentifiers] = useState({ Priority: [], Mapping: {} });
	const [eventSchema, setEventSchema] = useState(null);
	const [userSchema, setUserSchema] = useState(null);
	const [isLoading, setIsLoading] = useState(true);

	const { setTitle, api, showError, showStatus } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Anonymous IDs');
	}, []);

	useEffect(() => {
		const fetchData = async () => {
			let workspace;
			try {
				workspace = await api.workspace.get();
			} catch (err) {
				showError(err);
				return;
			}
			const transformed = transformAnonymousIdentifiers(workspace.AnonymousIdentifiers);
			setAnonymousIdentifiers(transformed);

			let eventSchema;
			try {
				eventSchema = await api.eventsSchema();
			} catch (err) {
				showError(err);
				return;
			}
			setEventSchema(eventSchema);

			let userSchema;
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
		const errorMessage = validateIdentifiersMapping(anonymousIdentifiers);
		if (errorMessage) {
			showError(errorMessage);
			return;
		}
		const untransformed = untransformAnonymousIdentifiers(anonymousIdentifiers);
		try {
			await api.workspace.anonymousIdentifiers(untransformed);
		} catch (err) {
			showError(err);
			return;
		}
		showStatus([variants.SUCCESS, icons.OK, 'Anonymous identifiers saved succesfully']);
	};

	return (
		<div className='anonymousIdentity'>
			{isLoading ? (
				<SlSpinner
					style={{
						fontSize: '3rem',
						'--track-width': '6px',
					}}
				></SlSpinner>
			) : (
				<Section
					title='Anonymous Identifiers'
					description='Define the identifiers used to resolve the identity of anonymous users'
				>
					<IdentifiersMapping
						mapping={anonymousIdentifiers}
						setMapping={setAnonymousIdentifiers}
						inputSchema={eventSchema}
						outputSchema={userSchema}
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
