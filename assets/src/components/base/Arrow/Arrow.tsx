import React, { ReactElement } from 'react';
import './Arrow.css';
import { ArrowAnchor } from './Arrow.types';
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
	showHead?: boolean;
	path?: 'smooth' | 'grid' | 'straight';
	label?: string | ReactElement;
	animateDrawing?: boolean;
}

const Arrow = ({
	start,
	end,
	startAnchor,
	endAnchor,
	dashness,
	color,
	isNew,
	showHead = false,
	path = 'smooth',
	label,
	animateDrawing = false,
}: ArrowProps) => {
	return (
		<div className={`arrow${isNew ? ' arrow--new' : ''}`}>
			<Xarrow
				start={start}
				end={end}
				startAnchor={startAnchor}
				endAnchor={endAnchor}
				showHead={showHead}
				color={color ? color : '#cacad6'}
				strokeWidth={1}
				curveness={0.7}
				dashness={dashness}
				path={path}
				headSize={8}
				tailSize={8}
				labels={label}
				animateDrawing={animateDrawing}
			/>
		</div>
	);
};

export default Arrow;
