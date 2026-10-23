import React, { useEffect, useLayoutEffect, useRef, useState } from 'react';
import 'react-date-range/dist/styles.css';
import 'react-date-range/dist/theme/default.css';
import { DateRange } from 'react-date-range';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaButtonGroup from '@awesome.me/webawesome/dist/react/button-group/index.js';
import { buttonVariant } from '../../../utils/buttonVariant';
import './SegmentedDateRangeControl.css';

const DATE_RANGE_INTERACTIVE_ELEMENTS = '.rdrCalendarWrapper button, .rdrCalendarWrapper input';

interface SegmentedDateRangePreset<T extends string> {
	value: T;
	label: string;
}

interface SegmentedDateRangeSelection {
	startDate: Date;
	endDate: Date;
	key: string;
}

interface SegmentedDateRangeControlProps<T extends string> {
	accessibleLabel?: string;
	presets: readonly SegmentedDateRangePreset<T>[];
	value: T | 'Custom';
	customRange: SegmentedDateRangeSelection[];
	onPresetChange: (preset: T) => void;
	onCustomRangeChange: (range: SegmentedDateRangeSelection[]) => void;
	maxDate?: Date;
	pickerAlignment?: 'start' | 'end';
	disabled?: boolean;
}

const formatCustomRange = (range: SegmentedDateRangeSelection[]): string => {
	const selection = range[0];
	if (selection == null) return 'Custom range';
	return `${selection.startDate.toLocaleDateString()} - ${selection.endDate.toLocaleDateString()}`;
};

const SegmentedDateRangeControl = <T extends string>({
	accessibleLabel,
	presets,
	value,
	customRange,
	onPresetChange,
	onCustomRangeChange,
	maxDate,
	pickerAlignment = 'start',
	disabled = false,
}: SegmentedDateRangeControlProps<T>) => {
	const root = useRef<HTMLDivElement>(null);
	const focusedPickerElement = useRef<number>();
	const [isPickerOpen, setIsPickerOpen] = useState<boolean>(false);
	const [pickerKey, setPickerKey] = useState<number>(0);

	useLayoutEffect(() => {
		const index = focusedPickerElement.current;
		if (index == null) return;
		focusedPickerElement.current = undefined;
		root.current?.querySelectorAll<HTMLElement>(DATE_RANGE_INTERACTIVE_ELEMENTS)[index]?.focus();
	}, [pickerKey]);

	useEffect(() => {
		const closeOnOutsideClick = (event: MouseEvent) => {
			if (event.target instanceof Node && !root.current?.contains(event.target)) {
				setIsPickerOpen(false);
			}
		};
		window.addEventListener('click', closeOnOutsideClick);
		return () => window.removeEventListener('click', closeOnOutsideClick);
	}, []);

	useEffect(() => {
		if (disabled) setIsPickerOpen(false);
	}, [disabled]);

	const selectPreset = (preset: T) => {
		setIsPickerOpen(false);
		onPresetChange(preset);
	};

	const selectCustomRange = (range: SegmentedDateRangeSelection) => {
		if (maxDate != null && (range.startDate > maxDate || range.endDate > maxDate)) {
			const elements = root.current?.querySelectorAll<HTMLElement>(DATE_RANGE_INTERACTIVE_ELEMENTS);
			const focusedIndex = Array.from(elements ?? []).findIndex((element) => element === document.activeElement);
			focusedPickerElement.current = focusedIndex === -1 ? undefined : focusedIndex;
			// react-date-range retains rejected editable values and selection focus internally.
			setPickerKey((key) => key + 1);
			return;
		}
		onCustomRangeChange([range]);
	};

	return (
		<div className='segmented-date-range-control' ref={root}>
			<WaButtonGroup label={accessibleLabel ?? 'Date range'}>
				{presets.map((preset) => (
					<WaButton
						key={preset.value}
						{...buttonVariant(value === preset.value ? 'primary' : 'default')}
						onClick={() => selectPreset(preset.value)}
						size='s'
						disabled={disabled}
					>
						{preset.label}
					</WaButton>
				))}
				<div className='segmented-date-range-control__custom'>
					<WaButton
						{...buttonVariant(value === 'Custom' ? 'primary' : 'default')}
						onClick={() => setIsPickerOpen((open) => !open)}
						size='s'
						disabled={disabled}
					>
						{value === 'Custom' ? formatCustomRange(customRange) : 'Custom range'}
					</WaButton>
					<div
						className={`segmented-date-range-control__picker segmented-date-range-control__picker--${pickerAlignment}${isPickerOpen ? ' segmented-date-range-control__picker--open' : ''}`}
					>
						<DateRange
							key={pickerKey}
							editableDateInputs={true}
							maxDate={maxDate}
							onChange={(item) => selectCustomRange(item.selection as SegmentedDateRangeSelection)}
							showSelectionPreview={true}
							moveRangeOnFirstSelection={false}
							months={2}
							ranges={customRange}
							direction='horizontal'
						/>
					</div>
				</div>
			</WaButtonGroup>
		</div>
	);
};

export type { SegmentedDateRangePreset, SegmentedDateRangeSelection };
export default SegmentedDateRangeControl;
