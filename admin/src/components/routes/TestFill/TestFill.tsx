import React, { useState } from 'react';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import './TestFill.css';

const TestFill = () => {
	const [nativeValue, setNativeValue] = useState<string>('');
	const [shoelaceValue, setShoelaceValue] = useState<string>('');

	const onChangeNativeValue = (e) => {
		setNativeValue(e.currentTarget.value);
	};

	const onChangeShoelaceValue = (e) => {
		setShoelaceValue(e.target.value);
	};

	return (
		<div className='test-fill'>
			<label>
				Input nativo:
				<input type='text' value={nativeValue} onChange={onChangeNativeValue} name='native-input' />
			</label>
			<SlInput
				label='Input di Shoelace:'
				type='text'
				value={shoelaceValue}
				onSlInput={onChangeShoelaceValue}
				name='shoelace-input'
			/>
		</div>
	);
};

export { TestFill };
