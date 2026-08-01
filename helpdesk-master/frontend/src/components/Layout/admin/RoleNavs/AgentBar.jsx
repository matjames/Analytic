import React from 'react'
import { Link } from 'react-router-dom'

const AgentBar = () => {
    return (
        <ul class="navbar-nav">
            <li class="nav-item dropdown">
                <Link class="nav-link dropdown-toggle arrow-none" to="/admin/dashboard" role="button">
                    <i class="bx bx-home-circle me-2"></i><span key="t-dashboards">Dashboard</span>
                </Link>
            </li>
            <li class="nav-item dropdown">
                <Link class="nav-link dropdown-toggle arrow-none" to="/admin/open/tickets" role="button">
                    <i class="bx bx-tone me-2"></i><span key="t-dashboards">Open Work Items</span>
                </Link>
            </li>
            <li class="nav-item dropdown">
                <Link class="nav-link dropdown-toggle arrow-none" to="/admin/inprogress/tickets" id="topnav-components" role="button"
                >
                    <i class="bx bx-collection me-2"></i><span key="t-components">In Progress</span>
                </Link>
            </li>
            <li class="nav-item dropdown">
                <Link class="nav-link dropdown-toggle arrow-none" to="/admin/closed/tickets" role="button">
                    <i class="bx bx-customize me-2"></i><span key="t-dashboards">Resolved Work Items</span>
                </Link>
            </li>
            <li class="nav-item dropdown">
                <Link class="nav-link dropdown-toggle arrow-none" to="/admin/overdue/tickets" id="topnav-components" role="button"
                >
                    <i class="bx bx-collection me-2"></i><span key="t-components">Overdue Items</span>
                </Link>
            </li>
        </ul>
    )
}

export default AgentBar