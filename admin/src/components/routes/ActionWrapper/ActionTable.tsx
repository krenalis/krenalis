import React, { useContext, useRef, useEffect } from 'react';
import Section from '../../shared/Section/Section';
import FeedbackButton from '../../shared/FeedbackButton/FeedbackButton';
import { AppContext } from '../../../context/providers/AppProvider';
import ActionContext from '../../../context/ActionContext';
import { CONFIRM_ANIMATION_DURATION } from './Action.constants';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { ObjectType } from '../../../types/external/types';
import { flattenSchema } from '../../../lib/helpers/transformedAction';

const ActionTable = () => {
	const { showError, api } = useContext(AppContext);
	const { connection, action, setAction, actionType, setActionType, mappingSectionRef, setIsTableChanged } =
		useContext(ActionContext);

	const tableConfirmationButtonRef = useRef<any>();
	const tableRef = useRef({
		lastConfirmation: '',
		lastUpdate: '',
	});

	useEffect(() => {
		tableRef.current = {
			lastConfirmation: action.Table!,
			lastUpdate: action.Table!,
		};
	}, []);

	const onUpdateTable = (e) => {
		const value = e.target.value;
		tableRef.current.lastUpdate = value;
		if (
			tableRef.current.lastUpdate !== tableRef.current.lastConfirmation &&
			tableRef.current.lastConfirmation !== ''
		) {
			setIsTableChanged(true);
		} else {
			setIsTableChanged(false);
		}
		const a = { ...action };
		a.Table = value;
		setAction(a);
	};

	const onConfirmTable = async () => {
		tableConfirmationButtonRef.current!.load();
		let schema: ObjectType;
		try {
			schema = await api.workspaces.connections.tableSchema(connection.id, action.Table);
		} catch (err) {
			tableConfirmationButtonRef.current!.stop();
			showError(err);
			return;
		}
		tableConfirmationButtonRef.current!.confirm();
		setTimeout(() => {
			tableRef.current.lastConfirmation = action.Table;
			const actionTyp = { ...actionType };
			actionTyp.OutputSchema = schema;
			setActionType(actionTyp);
			const a = { ...action };
			a.Mapping = flattenSchema(schema);
			setAction(a);
			setTimeout(() => {
				const top = mappingSectionRef.current!.getBoundingClientRect().top;
				mappingSectionRef.current!.closest('.fullscreen').scrollBy({
					top: top - 130,
					left: 0,
					behavior: 'smooth',
				});
			}, 100);
		}, CONFIRM_ANIMATION_DURATION);
	};

	return (
		<Section title='Table' description='The name of the table of the database'>
			<div className='actionTable'>
				<SlInput value={action.Table} onSlInput={onUpdateTable} />
				<FeedbackButton
					ref={tableConfirmationButtonRef}
					variant='success'
					size='small'
					onClick={onConfirmTable}
					animationDuration={CONFIRM_ANIMATION_DURATION}
				>
					Confirm
				</FeedbackButton>
			</div>
		</Section>
	);
};

export default ActionTable;
