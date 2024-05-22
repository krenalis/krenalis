import React, { useContext, useRef, useEffect } from 'react';
import Section from '../../base/Section/Section';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import AppContext from '../../../context/AppContext';
import ActionContext from '../../../context/ActionContext';
import { CONFIRM_ANIMATION_DURATION } from './Action.constants';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { ObjectType } from '../../../lib/api/types/types';
import { flattenSchema } from '../../../lib/helpers/transformedAction';
import { Popover } from '../../base/Popover/Popover';

const ActionTable = () => {
	const { handleError, api } = useContext(AppContext);
	const {
		connection,
		action,
		setAction,
		actionType,
		setActionType,
		transformationSectionRef,
		setIsTableChanged,
		isTransformationDisabled,
	} = useContext(ActionContext);

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
			handleError(err);
			return;
		}
		tableConfirmationButtonRef.current!.confirm();
		setTimeout(() => {
			tableRef.current.lastConfirmation = action.Table;
			setIsTableChanged(false);
			const actionTyp = { ...actionType };
			actionTyp.OutputSchema = schema;
			setActionType(actionTyp);
			const a = { ...action };
			a.Transformation.Mapping = flattenSchema(schema);
			setAction(a);
			setTimeout(() => {
				const top = transformationSectionRef.current!.getBoundingClientRect().top;
				transformationSectionRef.current!.closest('.fullscreen').scrollBy({
					top: top - 130,
					left: 0,
					behavior: 'smooth',
				});
			}, 100);
		}, CONFIRM_ANIMATION_DURATION);
	};

	return (
		<Section title='Table' description='The name of the table of the database'>
			<div className='action__table'>
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
				<Popover
					isOpen={isTransformationDisabled}
					content='Confirm when you have finished editing the table name.'
				/>
			</div>
		</Section>
	);
};

export default ActionTable;
