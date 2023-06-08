import './GridHeaderCell.css';

const GridHeaderCell = ({ value, alignment, className }) => {
	return (
		<div className={`${className}${value === '' ? ' empty' : ''}${alignment != null ? ` ${alignment}` : ''}`}>
			<div className='cellContent'>{value}</div>
		</div>
	);
};

export default GridHeaderCell;
