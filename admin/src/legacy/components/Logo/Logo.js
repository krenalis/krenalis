import React from 'react';
import { Link } from 'react-router-dom';
import './Logo.css';

export default class Logo extends React.Component {
    render() {
        return (
            <div className='Logo'>
                <div className='image'>C</div>
                <Link to='connectors'></Link>
            </div>
        )
    }
}
