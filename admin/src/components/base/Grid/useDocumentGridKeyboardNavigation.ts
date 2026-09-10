import { RefObject, useEffect } from 'react';
import { shouldNavigateGridFromDocument } from './GridKeyboardNavigation.helpers';
import { GridRef } from './Grid.types';

const useDocumentGridKeyboardNavigation = (gridRef: RefObject<GridRef>, enabled: boolean) => {
	useEffect(() => {
		if (!enabled) {
			return;
		}
		const onKeyDown = (event: KeyboardEvent) => {
			if (!shouldNavigateGridFromDocument(event)) {
				return;
			}
			const handled = gridRef.current?.navigate(event.key, event.shiftKey) === true;
			if (handled) {
				event.preventDefault();
				gridRef.current?.focus();
			}
		};
		document.addEventListener('keydown', onKeyDown);
		return () => document.removeEventListener('keydown', onKeyDown);
	}, [enabled, gridRef]);
};

export { useDocumentGridKeyboardNavigation };
