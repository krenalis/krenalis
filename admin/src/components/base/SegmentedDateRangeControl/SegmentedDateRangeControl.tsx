import React, { useEffect, useRef, useState } from 'react';
import 'react-date-range/dist/styles.css';
import 'react-date-range/dist/theme/default.css';
import { DateRange } from 'react-date-range';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlButtonGroup from '@shoelace-style/shoelace/dist/react/button-group/index.js';
import './SegmentedDateRangeControl.css';

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
	pickerAlignment?: 'start' | 'end';
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
	pickerAlignment = 'start',
}: SegmentedDateRangeControlProps<T>) => {
	const root = useRef<HTMLDivElement>(null);
	const [isPickerOpen, setIsPickerOpen] = useState<boolean>(false);

	useEffect(() => {
		const closeOnOutsideClick = (event: MouseEvent) => {
			if (event.target instanceof Node && !root.current?.contains(event.target)) {
				setIsPickerOpen(false);
			}
		};
		window.addEventListener('click', closeOnOutsideClick);
		return () => window.removeEventListener('click', closeOnOutsideClick);
	}, []);

	const selectPreset = (preset: T) => {
		setIsPickerOpen(false);
		onPresetChange(preset);
	};

	return (
		<div className='segmented-date-range-control' ref={root}>
			<SlButtonGroup label={accessibleLabel ?? 'Date range'}>
				{presets.map((preset) => (
					<SlButton
						key={preset.value}
						variant={value === preset.value ? 'primary' : 'default'}
						onClick={() => selectPreset(preset.value)}
						size='small'
					>
						{preset.label}
					</SlButton>
				))}
				<div className='segmented-date-range-control__custom'>
					<SlButton
						variant={value === 'Custom' ? 'primary' : 'default'}
						onClick={() => setIsPickerOpen((open) => !open)}
						size='small'
					>
						{value === 'Custom' ? formatCustomRange(customRange) : 'Custom range'}
					</SlButton>
					<div
						className={`segmented-date-range-control__picker segmented-date-range-control__picker--${pickerAlignment}${isPickerOpen ? ' segmented-date-range-control__picker--open' : ''}`}
					>
						<DateRange
							editableDateInputs={true}
							onChange={(item) => onCustomRangeChange([item.selection as SegmentedDateRangeSelection])}
							showSelectionPreview={true}
							moveRangeOnFirstSelection={false}
							months={2}
							ranges={customRange}
							direction='horizontal'
						/>
					</div>
				</div>
			</SlButtonGroup>
		</div>
	);
};

export type { SegmentedDateRangePreset, SegmentedDateRangeSelection };
export default SegmentedDateRangeControl;
