import { useState } from 'react';
import './GridNestedRows.css';
import GridRow from '../GridRow/GridRow';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const GridNestedRows = ({ rows, className }) => {
	let [isExpanded, setIsExpanded] = useState(false);

	let gridRows = [];
	for (let [i, cells] of rows.entries()) {
		if (i === 0) {
			gridRows.push(
				<>
					<SlIcon
						className='expand'
						name='caret-down-fill'
						onClick={() => {
							setIsExpanded(!isExpanded);
						}}
					></SlIcon>
					<GridRow cells={cells} className='GridRow parent' />
				</>
			);
			continue;
		}
		gridRows.push(<GridRow cells={cells} className='GridRow children' />);
	}

	return <div className={`${className}${isExpanded ? ' expanded' : ''}`}>{gridRows}</div>;
};

export default GridNestedRows;
