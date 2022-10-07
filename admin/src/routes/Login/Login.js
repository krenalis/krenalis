import React, { Component } from 'react'
import { Navigate } from 'react-router-dom'

import './Login.css'
import StatusMessage from '../../components/StatusMessage/StatusMessage';

export default class Login extends Component {
    constructor(props) {
        super(props);
        this.state = {
            'email': '',
            'password': '',
            'isLoggedIn': false,
            'statusMessage': '',
        }
    }

    handleLogin = async (e) => {
        e.preventDefault();
        this.setState({ statusMessage: '' })
        let customerID, error;
        try {
            let res = await fetch('/admin/', {
                method: 'POST',
                body: JSON.stringify({ email: this.state.email, password: this.state.password })
            });
            [customerID, error] = await res.json();
        } catch (err) {
            this.setState({ 'statusMessage': 'Something went wrong, check your connection and try again' });
            return;
        }
        if (error === 'AuthenticationFailedError') {
            this.setState({ 'statusMessage': 'Your email or password are incorrect' });
            return;
        }
        console.log(customerID);
        this.setState({ 'isLoggedIn': true });
    }

    onInputChange = (e) => {
        let name = e.currentTarget.name;
        let value = e.currentTarget.value;
        this.setState({
            [name]: value,
        });
    }

    render() {
        if (this.state.isLoggedIn) {
            return <Navigate to='connectors' />
        } else {
            return (
                <div className='Login'>
                    <div className='container'>
                        <div className='heading'>
                            <h1>Sign-in to your account</h1>
                        </div>
                        {this.state.statusMessage !== '' ? <StatusMessage text={this.state.statusMessage} onClose={() => { this.setState({ 'statusMessage': '' }) }} /> : ''}
                        <form className='form' onSubmit={this.handleLogin}>
                            <input type='text' onChange={this.onInputChange} name='email' value={this.state.text} placeholder='Your email' />
                            <input type='password' onChange={this.onInputChange} name='password' value={this.state.password} placeholder='Your password' />
                            <div className="note"><span>Note:</span> sign in with email <span>acme@open2b.com</span> and password <span>foopass2</span></div>
                            <input type='submit' value='Sign in' />
                        </form>
                    </div>
                </div>
            )
        }
    }
}
