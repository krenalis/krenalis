import React, { useState } from 'react';
import ReactDOM from 'react-dom/client';
import SegmentedDateRangeControl, {
	SegmentedDateRangeSelection,
} from '../src/components/base/SegmentedDateRangeControl/SegmentedDateRangeControl';

const PRESETS = [{ value: 'last7Days', label: 'Last 7 days' }] as const;

const formatDate = (date: Date): string => {
	const month = String(date.getMonth() + 1).padStart(2, '0');
	const day = String(date.getDate()).padStart(2, '0');
	return `${date.getFullYear()}-${month}-${day}`;
};

interface FixtureProps {
	id: string;
	maxDate?: Date;
}

const Fixture = ({ id, maxDate }: FixtureProps) => {
	const [range, setRange] = useState<SegmentedDateRangeSelection[]>([
		{
			startDate: new Date(2026, 7, 20),
			endDate: new Date(2026, 7, 25),
			key: 'selection',
		},
	]);

	return (
		<section data-testid={id}>
			<SegmentedDateRangeControl
				presets={PRESETS}
				value='Custom'
				customRange={range}
				onPresetChange={() => undefined}
				onCustomRangeChange={setRange}
				maxDate={maxDate}
			/>
			<output data-testid={`${id}-range`}>
				{formatDate(range[0].startDate)}:{formatDate(range[0].endDate)}
			</output>
		</section>
	);
};

const Harness = () => (
	<>
		<Fixture id='limited' maxDate={new Date(2026, 7, 25)} />
		<Fixture id='unlimited' />
	</>
);

ReactDOM.createRoot(document.getElementById('root')!).render(<Harness />);
