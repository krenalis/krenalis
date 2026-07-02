type ArrowAnchor =
	| 'middle'
	| 'left'
	| 'right'
	| 'top'
	| 'bottom'
	| 'auto'
	| {
			position: 'middle' | 'left' | 'right' | 'top' | 'bottom' | 'auto';
			offset: {
				x?: number;
				y?: number;
			};
	  };

export { ArrowAnchor };
