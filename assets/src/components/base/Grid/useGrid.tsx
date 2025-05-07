import React, { useState, useLayoutEffect } from 'react';
import { GridColumn, GridRow as GridRowType } from './Grid.types';

const useGrid = (
	gridRef: React.MutableRefObject<any>,
	rows: GridRowType[],
	columns: GridColumn[],
	gridColumns?: string,
	isLoading?: boolean,
	isShown?: boolean,
) => {
	const [columnsWidths, setColumnsWidths] = useState<string>();
	const [reloadWidths, setReloadWidths] = useState<boolean>();

	useLayoutEffect(() => {
		const computeColumnsWidths = () => {
			const widthsOfColumn = {};
			for (let i = 0; i < columns.length; i++) {
				widthsOfColumn[i] = [];
			}
			const rowElements = gridRef.current.querySelectorAll('.grid__header-row, .grid__row');
			for (const r of rowElements) {
				const contents = r.querySelectorAll('.grid__cell-content');
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
		};
		if (gridColumns != null) {
			return;
		}
		if (isLoading || (isShown != null && !isShown)) {
			return;
		}
		if (gridRef.current == null) {
			return;
		}
		if (reloadWidths === true) {
			// reset the boolean and trigger a new computation of the widths.
			setReloadWidths(null);
			return;
		}
		setTimeout(computeColumnsWidths);
	}, [rows, columns, isLoading, isShown, reloadWidths]);

	return { columnsWidths, reloadColumnsWidths: () => setReloadWidths(true) };
};

export { useGrid };
