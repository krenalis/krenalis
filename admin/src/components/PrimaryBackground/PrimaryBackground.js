import './PrimaryBackground.css';

const PrimaryBackground = ({ children, height, overlap, contentWidth }) => {
	let style = {};
	if (height) style['height'] = `${height}px`;
	if (overlap) style['marginBottom'] = `-${overlap}px`;
	let wrapperStyle;
	if (contentWidth) wrapperStyle = { width: `${contentWidth}px`, margin: 'auto' };
	return (
		<div className='PrimaryBackground' style={style}>
			<div className='widthWrapper' style={wrapperStyle}>
				{children}
			</div>
		</div>
	);
};

export default PrimaryBackground;
