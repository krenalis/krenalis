import React, { Component } from 'react'
import { Link } from 'react-router-dom'
import './Home.css'

export default class Home extends Component {
    render() {
        return (
            <div className="Home">
                <div className="content">
                    <div className="title">
                        <h1>Chichi</h1>
                        <p>Seleziona una versione della dashboard</p>
                    </div>
                    <div className="buttons">
                        <a class="btn vanilla" href="visualization">Vanilla dashboard</a>
                        <Link class="btn react" to="dashboard">React dashboard</Link>
                    </div>
                </div>
            </div>
        )
    }
}
