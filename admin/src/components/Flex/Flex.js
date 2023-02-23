import './Flex.css';

const Flex = ({ className, justifyContent, alignItems, gap, children }) => {
	return (
		<div
			className={`flex${className != null ? ` ${className}` : ''}`}
			style={{ justifyContent: justifyContent, alignItems: alignItems, gap: gap }}
		>
			{children}
		</div>
	);
};

export default Flex;
