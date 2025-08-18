import React, { ReactElement } from 'react';
import './Arrow.css';
import { ArrowAnchor } from './Arrow.types';
import Xarrow from 'react-xarrows';

interface ArrowProps {
	start: string;
	end: string;
	startAnchor: ArrowAnchor;
	endAnchor: ArrowAnchor;
	strokeWidth?: number;
	curveness?: number;
	dashness?:
		| boolean
		| {
				strokeLen?: number;
				nonStrokeLen?: number;
				animation?: boolean | number;
		  };
	color?: string;
	isNew?: boolean;
	path?: 'smooth' | 'grid' | 'straight';
	label?: string | ReactElement;
	animateDrawing?: boolean;
	isHidden?: boolean;
	showTail?: boolean;
	showHead?: boolean;
}

const Arrow = ({
	start,
	end,
	startAnchor,
	endAnchor,
	curveness,
	dashness,
	strokeWidth,
	color,
	isNew,
	path = 'smooth',
	label,
	animateDrawing = false,
	isHidden = false,
	showTail = false,
	showHead = false,
}: ArrowProps) => {
	const shape = {
		svgElem: (
			<path
				d='M 1,0 V 1.0000008 A 0.81233699,0.50003097 0 0 1 0.18813242,0.50025884 0.81233699,0.50003097 0 0 1 1,0 Z'
				style={{ fill: color ? color : '#B9B9CA' }}
			/>
		),
	};

	return (
		<div className={`arrow${isNew ? ' arrow--new' : ''}${isHidden ? ' arrow--hidden' : ''}`}>
			<Xarrow
				start={start}
				end={end}
				startAnchor={startAnchor}
				endAnchor={endAnchor}
				color={color ? color : '#B9B9CA'}
				strokeWidth={strokeWidth ? strokeWidth : 1}
				curveness={curveness ? curveness : 0.7}
				dashness={dashness}
				path={path}
				headSize={6}
				tailSize={6}
				labels={label}
				animateDrawing={animateDrawing}
				showTail={showTail}
				showHead={showHead}
				tailShape={shape}
				headShape={shape}
			/>
		</div>
	);
};

export default Arrow;
