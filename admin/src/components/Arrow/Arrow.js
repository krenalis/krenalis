import './Arrow.css';
import Xarrow from 'react-xarrows';

const Arrow = ({ start, end, startAnchor, endAnchor, isNew }) => {
	return (
		<div className={`Arrow${isNew ? ' new' : ''}`}>
			<Xarrow
				start={start}
				end={end}
				startAnchor={startAnchor}
				endAnchor={endAnchor}
				showHead={false}
				color='#cacad6'
				strokeWidth={1}
			/>
		</div>
	);
};

export default Arrow;
