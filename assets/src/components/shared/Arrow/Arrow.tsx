import React from 'react';
import './Arrow.css';
import { ArrowAnchor } from '../../../types/internal/app';
import Xarrow from 'react-xarrows';

interface ArrowProps {
	start: string;
	end: string;
	startAnchor: ArrowAnchor;
	endAnchor: ArrowAnchor;
	dashness?:
		| boolean
		| {
				strokeLen?: number;
				nonStrokeLen?: number;
				animation?: boolean | number;
		  };
	color?: string;
	isNew?: boolean;
}

const Arrow = ({ start, end, startAnchor, endAnchor, dashness, color, isNew }: ArrowProps) => {
	return (
		<div className={`arrow${isNew ? ' arrow--new' : ''}`}>
			<Xarrow
				start={start}
				end={end}
				startAnchor={startAnchor}
				endAnchor={endAnchor}
				showHead={false}
				color={color ? color : '#cacad6'}
				strokeWidth={1}
				curveness={0.7}
				dashness={dashness}
			/>
		</div>
	);
};

export default Arrow;
