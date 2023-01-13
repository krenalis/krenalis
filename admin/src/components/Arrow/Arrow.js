import './Arrow.css';
import Xarrow from 'react-xarrows';

const Arrow = ({ start, end, startAnchor, endAnchor }) => {
	return (
		<div className='Arrow'>
			<Xarrow
				start={start}
				end={end}
				startAnchor={startAnchor}
				endAnchor={endAnchor}
				showHead={false}
				color='#a1a1aa'
				strokeWidth={2}
			/>
		</div>
	);
};

export default Arrow;
