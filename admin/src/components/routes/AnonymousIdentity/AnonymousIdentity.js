import { useEffect, useLayoutEffect, useState } from 'react';
import './AnonymousIdentity.css';
import SortableMapping from '../../common/SortableMapping/SortableMapping';
import Section from '../../common/Section/Section';
import * as variants from '../../../constants/variants';
import * as icons from '../../../constants/icons';
import { useContext } from 'react';
import { AppContext } from '../../../providers/AppProvider';
import useTransformedAnonymousIdentifiers from '../../../hooks/useTransformedAnonymousIdentifiers';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

const AnonymousIdentity = () => {
	const [anonymousIdentifiers, setAnonymousIdentifiers] = useState({ Priority: [], Mapping: {} });
	const [eventSchema, setEventSchema] = useState(null);
	const [userSchema, setUserSchema] = useState(null);

	const { setTitle, api, showError, showStatus } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Anonymous IDs');
	}, []);

	useEffect(() => {
		const fetchData = async () => {
			let workspace, err;
			[workspace, err] = await api.workspace.get();
			if (err) {
				showError(err);
				return;
			}
			setAnonymousIdentifiers(workspace.AnonymousIdentifiers);

			let eventSchema;
			[eventSchema, err] = await api.eventsSchema();
			if (err) {
				showError(err);
				return;
			}
			setEventSchema(eventSchema);

			let userSchema;
			[userSchema, err] = await api.workspace.userSchema();
			if (err) {
				showError(err);
				return;
			}
			setUserSchema(userSchema);
		};
		fetchData();
	}, []);

	const onSave = async () => {
		const [, err] = await api.workspace.anonymousIdentifiers(anonymousIdentifiers);
		if (err) {
			showError(err);
			return;
		}
		showStatus([variants.SUCCESS, icons.OK, 'Anonymous identifiers saved succesfully']);
	};

	const { transformedAnonymousIdentifiers, setTransformedAnonymousIdentifiers } = useTransformedAnonymousIdentifiers(
		anonymousIdentifiers,
		setAnonymousIdentifiers
	);

	return (
		<div className='anonymousIdentity'>
			<Section
				title='Anonymous Identifiers'
				description='Define the identifiers used to resolve the identity of anonymous users'
			>
				<SortableMapping
					mapping={transformedAnonymousIdentifiers}
					setMapping={setTransformedAnonymousIdentifiers}
					inputSchema={eventSchema}
					outputSchema={userSchema}
				/>
				<SlButton className='saveButton' onClick={onSave} variant='primary'>
					Save
				</SlButton>
			</Section>
		</div>
	);
};

export default AnonymousIdentity;
