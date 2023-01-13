import './FlexContainer.css';

const FlexContainer = ({ className, justifyContent, alignItems, gap, children }) => {
	return (
		<div
			className={`FlexContainer${className != null ? ` ${className}` : ''}`}
			style={{ justifyContent: justifyContent, alignItems: alignItems, gap: gap }}
		>
			{children}
		</div>
	);
};

export default FlexContainer;
