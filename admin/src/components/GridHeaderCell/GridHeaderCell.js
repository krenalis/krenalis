import './GridHeaderCell.css';

const GridHeaderCell = ({ value, className }) => {
	return (
		<div className={className}>
			<div className='cellContent'>{value}</div>
		</div>
	);
};

export default GridHeaderCell;
