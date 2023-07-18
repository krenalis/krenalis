import './Arrow.css';
import Xarrow from 'react-xarrows';

const Arrow = ({ start, end, startAnchor, endAnchor, dashness, color, isNew }) => {
	return (
		<div className={`arrow${isNew ? ' new' : ''}`}>
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
