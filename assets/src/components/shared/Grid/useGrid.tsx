import React, { useState, useLayoutEffect } from 'react';
import { GridColumn, GridRow as GridRowType } from '../../../types/componentTypes/Grid.types';

const useGrid = (
	gridRef: React.MutableRefObject<any>,
	rows: GridRowType[],
	columns: GridColumn[],
	isLoading?: boolean,
	isShown?: boolean,
) => {
	const [columnsWidths, setColumnsWidths] = useState<string>();

	useLayoutEffect(() => {
		if (isLoading || (isShown != null && !isShown)) {
			return;
		}
		if (gridRef.current == null) {
			return;
		}
		const widthsOfColumn = {};
		for (let i = 0; i < columns.length; i++) {
			widthsOfColumn[i] = [];
		}
		const rowElements = gridRef.current.querySelectorAll('.gridHeaderRow, .gridRow');
		for (const r of rowElements) {
			const contents = r.querySelectorAll('.cellContent');
			for (const [i, c] of contents.entries()) {
				if (c instanceof HTMLElement) {
					widthsOfColumn[i].push(c.offsetWidth);
				}
			}
		}
		const maxWidths = [] as number[];
		for (const k in widthsOfColumn) {
			const widths = widthsOfColumn[k];
			maxWidths.push(Math.max(...widths) + 40); // 40 is the left/right padding of the cells.
		}
		let widths = '';
		for (let i = 0; i < maxWidths.length; i++) {
			if (i === 0) {
				widths += `${maxWidths[i]}px`;
			} else {
				widths += ` ${maxWidths[i]}px`;
			}
		}
		setColumnsWidths(widths);
	}, [rows, columns, isLoading, isShown]);

	return { columnsWidths };
};

export { useGrid };
