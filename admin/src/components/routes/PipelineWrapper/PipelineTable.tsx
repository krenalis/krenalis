import React, { useContext, useRef, useEffect, useMemo } from 'react';
import Section from '../../base/Section/Section';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import AppContext from '../../../context/AppContext';
import PipelineContext from '../../../context/PipelineContext';
import { CONFIRM_ANIMATION_DURATION, ERROR_ANIMATION_DURATION } from './Pipeline.constants';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { flattenSchema, propertyTypesAreEqual } from '../../../lib/core/pipeline';
import { Popover } from '../../base/Popover/Popover';
import { getTableKeyComboboxItems } from '../../helpers/getSchemaComboboxItems';
import { Combobox } from '../../base/Combobox/Combobox';
import { TableSchemaResponse } from '../../../lib/api/types/responses';

const PipelineTable = () => {
	const { handleError, api } = useContext(AppContext);
	const {
		connection,
		pipeline,
		setPipeline,
		pipelineType,
		setPipelineType,
		transformationSectionRef,
		setIsTableChanged,
		isTransformationDisabled,
		isTransformationHidden,
		setIssues,
	} = useContext(PipelineContext);

	const tableConfirmationButtonRef = useRef<any>();
	const tableKeySectionRef = useRef<any>();
	const tableKeyRef = useRef<any>();
	const tableRef = useRef({
		lastConfirmation: '',
		lastUpdate: '',
	});

	useEffect(() => {
		tableRef.current = {
			lastConfirmation: pipeline.tableName!,
			lastUpdate: pipeline.tableName!,
		};
	}, []);

	const tableKeyComboboxItems = useMemo(() => {
		return getTableKeyComboboxItems(pipelineType.outputSchema);
	}, [pipelineType]);

	const onUpdateTableName = (e) => {
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
		const p = { ...pipeline };
		p.tableName = value;
		setPipeline(p);
	};

	const onTableKeyPropertyUpdate = (_: string, value: string) => {
		const p = { ...pipeline };
		p.tableKey = value;
		setPipeline(p);
	};

	const onTableKeyPropertySelect = (_: string, value: string) => {
		const p = { ...pipeline };
		p.tableKey = value;
		setPipeline(p);
	};

	const onConfirmTable = async () => {
		setIssues([]);
		tableConfirmationButtonRef.current!.load();
		let res: TableSchemaResponse;
		try {
			res = await api.workspaces.connections.tableSchema(connection.id, pipeline.tableName);
		} catch (err) {
			tableConfirmationButtonRef.current!.stop();
			handleError(err);
			return;
		}
		if (res.schema == null) {
			tableConfirmationButtonRef.current!.error("This table doesn't have any compatible column");
			setTimeout(() => {
				setIssues(res.issues);
				const pipelineTyp = { ...pipelineType };
				pipelineTyp.outputSchema = null;
				setPipelineType(pipelineTyp);
			}, ERROR_ANIMATION_DURATION);
		} else {
			tableConfirmationButtonRef.current!.confirm();
			setTimeout(() => {
				setIssues(res.issues);
				tableRef.current.lastConfirmation = pipeline.tableName;
				setIsTableChanged(false);
				const pipelineTyp = { ...pipelineType };
				pipelineTyp.outputSchema = res.schema;
				setPipelineType(pipelineTyp);
				const p = { ...pipeline };
				const mapping = flattenSchema(res.schema, true);
				if (p.transformation.mapping != null) {
					// Keep the old mapping (if the column stil exists
					// in the new out schema and the type is the same).
					for (const path in mapping) {
						const existedInOldSchema = p.transformation.mapping[path] != null;
						if (!existedInOldSchema) {
							continue;
						}
						const newType = mapping[path].full.type;
						const oldType = p.transformation.mapping[path].full.type;
						if (!propertyTypesAreEqual(newType, oldType)) {
							continue;
						}
						mapping[path].value = p.transformation.mapping[path].value;
						mapping[path].error = p.transformation.mapping[path].error;
					}
				}
				p.transformation.mapping = mapping;
				setPipeline(p);
				setTimeout(() => {
					let scrollSection = transformationSectionRef.current;
					if (tableKeyRef.current != null) {
						scrollSection = tableKeySectionRef.current;
					}
					const top = scrollSection.getBoundingClientRect().top;
					scrollSection.closest('.fullscreen').scrollBy({
						top: top - 130,
						left: 0,
						behavior: 'smooth',
					});
				}, 100);
			}, CONFIRM_ANIMATION_DURATION);
		}
	};

	return (
		<>
			<Section
				title='Destination table'
				description='Enter the name of an existing database table where profile data will be exported.'
				padded={true}
				annotated={true}
			>
				<div className='pipeline__destination_table'>
					<SlInput value={pipeline.tableName} onSlInput={onUpdateTableName} />
					<FeedbackButton
						ref={tableConfirmationButtonRef}
						variant='success'
						size='small'
						onClick={onConfirmTable}
						animationDuration={CONFIRM_ANIMATION_DURATION}
						disabled={pipeline.tableName === ''}
					>
						Confirm
					</FeedbackButton>
					<Popover
						isOpen={isTransformationDisabled}
						content='Confirm when you have finished editing the table name.'
					/>
				</div>
			</Section>
			{pipelineType.target === 'User' && !isTransformationHidden && (
				<Section
					title='Matching column'
					description='Select the column used to match table records for updates, or to create a new record when no match is found.'
					padded={true}
					annotated={true}
					ref={tableKeySectionRef}
					className={`pipeline__destination_table-key-section${isTransformationDisabled ? ' pipeline__destination_table-key-section--disabled' : ''}`}
				>
					<div className='pipeline__destination_table-key-property' ref={tableKeyRef}>
						<Combobox
							value={pipeline.tableKey}
							onInput={onTableKeyPropertyUpdate}
							name='table-key'
							items={tableKeyComboboxItems}
							onSelect={onTableKeyPropertySelect}
							disabled={isTransformationDisabled}
							isExpression={false}
						/>
					</div>
				</Section>
			)}
		</>
	);
};

export default PipelineTable;
