import React, { useEffect, useState, ReactNode, Fragment } from 'react';
import './GridNestedRows.css';
import GridRow from '../GridRow/GridRow';
import { NestedGridRows, GridColumn, GridNestedRowsIndentation, SortableGridRow } from '../Grid.types';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { SortableRows } from '../SortableRows';

interface GridNestedRowsProps {
	rows: NestedGridRows;
	columns: GridColumn[];
	className?: string;
	nesting: number;
	onSortRow?: (overRowID: string, movedRowID: string) => void;
	isSortable?: boolean;
	reorderDisabled?: boolean;
	indentation?: GridNestedRowsIndentation;
	reloadColumnsWidths: () => void;
}

const GridNestedRows = ({
	rows,
	columns,
	className,
	nesting,
	onSortRow,
	isSortable,
	reorderDisabled,
	indentation,
	reloadColumnsWidths,
}: GridNestedRowsProps) => {
	const rootRow = Array.isArray(rows[0]) ? null : rows[0];
	const [isExpanded, setIsExpanded] = useState(rootRow?.expanded === true);
	const forceExpanded = rootRow?.forceExpanded === true;

	const onToggleExpansion = (event: React.MouseEvent, onSelect?: () => void) => {
		if (!forceExpanded) {
			setIsExpanded(!isExpanded);
			reloadColumnsWidths();
		}
		// Expand and collapse all use programmatic clicks and must not change the selected row.
		if (event.isTrusted) {
			onSelect?.();
		}
	};

	useEffect(() => {
		if (rootRow?.expanded === true) {
			setIsExpanded(true);
		}
	}, [rootRow?.expanded]);

	let parentComponent: ReactNode = null;
	let childrenComponents: any[] = [];
	for (const [i, row] of rows.entries()) {
		if (Array.isArray(row)) {
			const nestedRows = row as NestedGridRows;
			const parentRow = nestedRows[0] as SortableGridRow;
			const component = (
				<GridNestedRows
					key={parentRow.id ?? i}
					rows={nestedRows}
					columns={columns}
					className='grid__nested-rows grid__nested-rows--children'
					nesting={nesting + 1}
					onSortRow={onSortRow}
					isSortable={isSortable}
					reorderDisabled={reorderDisabled}
					indentation={indentation}
					reloadColumnsWidths={reloadColumnsWidths}
				/>
			);
			if (isSortable) {
				childrenComponents.push({
					id: parentRow.dragKey,
					row: component,
				});
			} else {
				childrenComponents.push(component);
			}
		} else {
			const r = row as any;
			if (i === 0) {
				parentComponent = (
					<Fragment key={i}>
						<SlIcon
							className='grid__row-expand'
							name='caret-right-fill'
							onClick={(event) => onToggleExpansion(event, r.onToggleExpansion ?? r.onClick)}
						></SlIcon>
						<GridRow row={r} columns={columns} className='grid__row grid__row--parent' />
					</Fragment>
				);
			} else {
				const component = (
					<GridRow key={i} row={r} columns={columns} className='grid__row grid__row--children' />
				);
				if (isSortable) {
					const r = row as SortableGridRow;
					childrenComponents.push({
						id: r.dragKey,
						row: component,
					});
				} else {
					childrenComponents.push(component);
				}
			}
		}
	}

	const baseIndentation = indentation?.base ?? 50;
	const indentationStep = indentation?.step ?? 30;
	const parentIndentation = baseIndentation + indentationStep * (nesting - 1) + 'px';
	const childrenIndentation = baseIndentation + indentationStep * nesting + 'px';
	return (
		<div
			className={`${className}${forceExpanded || isExpanded ? ' grid__nested-rows--expanded' : ''}`}
			style={
				{
					'--parent-indentation': parentIndentation,
					'--children-indentation': childrenIndentation,
				} as React.CSSProperties
			}
		>
			{parentComponent}
			{isSortable ? (
				<SortableRows disabled={reorderDisabled} onSortRow={onSortRow}>
					{childrenComponents}
				</SortableRows>
			) : (
				childrenComponents
			)}
		</div>
	);
};

export default GridNestedRows;
