/* eslint-disable jsx-a11y/anchor-is-valid */
import React from 'react'
import AdminBar from './RoleNavs/AdminBar'
import AgentBar from './RoleNavs/AgentBar'

const TopNavBar = () => {

    const user = JSON.parse(localStorage.getItem("user"))

    return (
        <div class="topnav">
            <div class="container-fluid">
                <nav class="navbar navbar-light navbar-expand-lg topnav-menu">
                    <div class="collapse navbar-collapse" id="topnav-menu-content">
                        {user && user.role === 'admin' && <AdminBar />}
                        {user && user.role === 'agent' && <AgentBar />}
                    </div>
                </nav>
            </div>
        </div>
    )
}

export default TopNavBar